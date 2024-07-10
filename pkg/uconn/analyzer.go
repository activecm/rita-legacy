package uconn

import (
	"sync"

	"github.com/activecm/rita-legacy/config"
	"github.com/activecm/rita-legacy/database"
	"github.com/activecm/rita-legacy/pkg/data"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (
	//analyzer records data regarding the connections between pairs of hosts
	analyzer struct {
		chunk            int                        //current chunk (0 if not on rolling analysis)
		connLimit        int64                      // limit for strobe classification
		db               *database.DB               // provides access to MongoDB
		log              *log.Logger                // logger for writing out errors and warnings
		conf             *config.Config             // contains details needed to access MongoDB
		analyzedCallback func(database.BulkChanges) // called on each analyzed result
		closedCallback   func()                     // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Input                // holds unanalyzed data
		analysisWg       sync.WaitGroup             // wait for analysis to finish
	}
)

// newAnalyzer creates a new analyzer for recording unique connection records
func newAnalyzer(chunk int, connLimit int64, db *database.DB, log *log.Logger, conf *config.Config, analyzedCallback func(database.BulkChanges), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		connLimit:        connLimit,
		db:               db,
		log:              log,
		conf:             conf,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *Input),
	}
}

// collect gathers unique connection records for analysis
func (a *analyzer) collect(datum *Input) {
	a.analysisChannel <- datum
}

// close waits for the analyzer to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

// start kicks off a new analysis thread
func (a *analyzer) start() {
	a.analysisWg.Add(1)
	go func() {
		ssn := a.db.Session.Copy()
		defer ssn.Close()

		uconnColl := ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.UniqueConnTable)

		for datum := range a.analysisChannel {

			// mainQuery handles setting the unique connection src, dst, and strobe status.
			// Additionally, mainQuery formats a `dat` subdocument which represents the
			// connection statistics for this chunk of imports.
			mainUpdate := mainQuery(datum, a.connLimit, a.chunk)

			// openConnectionsQuery handles summarizing any open connections and formats an update
			// to the top-level open connection fields in the unique connection doc
			openConnsUpdate, openCount, openTBytes, openDur := openConnectionsQuery(datum)

			// rollUpQuery aggregates any existing `dat` subdocuments representing the connection
			// statistics together with the statistics for the current chunk of imports and the
			// the open connection statistics. Then, the rollUpQuery formats an update to the
			// top-level connection statistics fields in the unique connection doc.
			rollUpUpdate, err := rollUpQuery(datum, openCount, openTBytes, openDur, uconnColl)
			if err != nil {
				a.log.WithFields(log.Fields{
					"Module": "host",
					"Data":   datum.Hosts.BSONKey(),
				}).Error(err)
			}

			totalUpdate := database.MergeBSONMaps(mainUpdate, openConnsUpdate, rollUpUpdate)

			a.analyzedCallback(database.BulkChanges{
				a.conf.T.Structure.UniqueConnTable: []database.BulkChange{{
					Selector: datum.Hosts.BSONKey(),
					Update:   totalUpdate,
					Upsert:   true,
				}},
			})
		}
		a.analysisWg.Done()
	}()
}

// mainQuery records the bulk of the information about the communications between two hosts
func mainQuery(datum *Input, strobeLimit int64, chunk int) bson.M {

	// Truncate the protocol/ port tuples we store in the database
	tuples := datum.Tuples.Items()
	if len(tuples) > 5 {
		tuples = tuples[:5]
	}

	// if this connection qualifies to be a strobe with the current number
	// of connections in the current datum, don't store bytes and ts.
	// it will not qualify to be downgraded to a beacon until this chunk is
	// outdated and removed. If only importing once - still just a strobe.
	ts := datum.TsList
	bytes := datum.OrigBytesList

	isStrobe := datum.ConnectionCount >= strobeLimit
	if isStrobe {
		ts = []int64{}
		bytes = []int64{}
	}

	return bson.M{
		"$set": bson.M{
			// strobe status must be set/unset in uconns so that we avoid querying
			// uconns that are definitely strobes during beacon analysis
			"strobe":           isStrobe,
			"cid":              chunk,
			"src_network_name": datum.Hosts.SrcNetworkName,
			"dst_network_name": datum.Hosts.DstNetworkName,
		},
		"$push": bson.M{
			"dat": bson.M{
				"$each": []bson.M{{
					"count":  datum.ConnectionCount,
					"bytes":  bytes,
					"ts":     ts,
					"tuples": tuples,
					"icerts": datum.InvalidCertFlag,
					"maxdur": datum.MaxDuration,
					"tbytes": datum.TotalBytes,
					"tdur":   datum.TotalDuration,
					"cid":    chunk,
				}},
			},
		},
	}
}

// openConnectionsQuery records information about connections that are still open between two hosts.
// Additionally this function returns the summary details of the open connections
func openConnectionsQuery(datum *Input) (query bson.M, connCount, totalBytes int64, duration float64) {
	var origBytes int64
	tsList := make([]int64, 0)

	// Tally up the bytes and duration from the open connections.
	// We will add these at the top level of the current uconn entry
	// when it's placed in mongo such that we will have an up-to-date
	// total for open connection values each time we parse another
	// set of logs. These current values will overwrite any existing values.
	// The relevant values from the closed connection will be added to the
	// appropriate chunk in a "dat" and those values will effectively be
	// removed from the open connection values that we are tracking.
	for key, connStateEntry := range datum.ConnStateMap {
		if connStateEntry.Open {
			totalBytes += connStateEntry.Bytes
			duration += connStateEntry.Duration
			origBytes += connStateEntry.OrigBytes

			connCount++

			// Only append unique timestamps to OpenTsList.
			if !int64InSlice(connStateEntry.Ts, tsList) {
				tsList = append(tsList, connStateEntry.Ts)
			}
		} else {
			// Remove the closed entry so it doesn't appear in the list of open connections in mongo
			// Interwebs says it is safe to do this operation within a range loop
			// source: https://stackoverflow.com/questions/23229975/is-it-safe-to-remove-selected-keys-from-map-within-a-range-loop
			// This will also prevent duplication of data between a previously-opened and closed connection that are
			// one in the same
			delete(datum.ConnStateMap, key)
		}
	}

	connState := len(datum.ConnStateMap) > 0

	query = bson.M{
		"$set": bson.M{
			"open":                  connState,
			"open_bytes":            totalBytes,
			"open_connection_count": connCount,
			"open_conns":            datum.ConnStateMap,
			"open_duration":         duration,
			"open_orig_bytes":       origBytes,
			"open_ts":               tsList,
		},
	}
	return
}

// rollUpQuery updates the top level summary fields which aggregate over rhe chunked fields
func rollUpQuery(datum *Input, openCount, openTotalBytes int64, openDuration float64, uconnColl *mgo.Collection) (bson.M, error) {
	//existingQuery gathers the previously existing summary details for the connection pair
	existingQuery := []bson.M{
		{"$match": datum.Hosts.BSONKey()},
		{"$project": bson.M{
			"count": "$dat.count",
			"tuples": bson.M{
				"$ifNull": []interface{}{"$dat.tuples", []interface{}{}},
			},
			"tbytes": "$dat.tbytes",
			"tdur":   "$dat.tdur",
		}},
		{"$unwind": "$count"},
		{"$group": bson.M{
			"_id":    nil,
			"count":  bson.M{"$sum": "$count"},
			"tuples": bson.M{"$first": "$tuples"},
			"tbytes": bson.M{"$first": "$tbytes"},
			"tdur":   bson.M{"$first": "$tdur"},
		}},
		{"$unwind": "$tuples"},
		{"$unwind": "$tuples"}, // not an error, must be done twice
		{"$group": bson.M{
			"_id":    nil,
			"count":  bson.M{"$first": "$count"},
			"tuples": bson.M{"$addToSet": "$tuples"},
			"tbytes": bson.M{"$first": "$tbytes"},
			"tdur":   bson.M{"$first": "$tdur"},
		}},
		{"$unwind": "$tbytes"},
		{"$group": bson.M{
			"_id":    nil,
			"count":  bson.M{"$first": "$count"},
			"tuples": bson.M{"$first": "$tuples"},
			"tbytes": bson.M{"$sum": "$tbytes"},
			"tdur":   bson.M{"$first": "$tdur"},
		}},
		{"$unwind": "$tdur"},
		{"$group": bson.M{
			"_id":    nil,
			"count":  bson.M{"$first": "$count"},
			"tuples": bson.M{"$first": "$tuples"},
			"tbytes": bson.M{"$first": "$tbytes"},
			"tdur":   bson.M{"$sum": "$tdur"},
		}},
	}
	type rollUpResult struct {
		Count         int64    `bson:"count"`
		Tuples        []string `bson:"tuples"`
		TotalBytes    int64    `bson:"tbytes"`
		TotalDuration float64  `bson:"tdur"`
	}
	var rollUpRes rollUpResult
	err := uconnColl.Pipe(existingQuery).AllowDiskUse().One(&rollUpRes)
	if err != nil && err != mgo.ErrNotFound {
		return bson.M{}, err
	}

	// add in the open connection stats
	rollUpRes.Count += openCount
	rollUpRes.TotalBytes += openTotalBytes
	rollUpRes.TotalDuration += openDuration

	// add in the current data

	// merge tuples and limit to 5
	const maxTuples = 5

	tupleSet := make(data.StringSet)
	for _, tuple := range rollUpRes.Tuples {
		tupleSet.Insert(tuple)
	}
	for tuple := range datum.Tuples {
		tupleSet.Insert(tuple)
	}
	rollUpRes.Tuples = tupleSet.Items()

	if len(rollUpRes.Tuples) > maxTuples {
		rollUpRes.Tuples = rollUpRes.Tuples[:5]
	}

	rollUpRes.Count += datum.ConnectionCount
	rollUpRes.TotalBytes += datum.TotalBytes
	rollUpRes.TotalDuration += datum.TotalDuration

	return bson.M{
		"$set": bson.M{
			"count":  rollUpRes.Count,
			"tuples": rollUpRes.Tuples,
			"tbytes": rollUpRes.TotalBytes,
			"tdur":   rollUpRes.TotalDuration,
		},
	}, nil
}

// int64InSlice checks if a given int64 is in a slice
func int64InSlice(a int64, list []int64) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
