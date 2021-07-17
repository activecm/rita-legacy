package uconn

import (
	"strconv"
	"sync"

	"github.com/activecm/rita/pkg/data"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for exploded dns analysis
	analyzer struct {
		chunk            int            //current chunk (0 if not on rolling analysis)
		chunkStr         string         //current chunk (0 if not on rolling analysis)
		connLimit        int64          // limit for strobe classification
		db               *database.DB   // provides access to MongoDB
		conf             *config.Config // contains details needed to access MongoDB
		analyzedCallback func(update)   // called on each analyzed result
		closedCallback   func()         // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Input    // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for parsing hostnames
func newAnalyzer(chunk int, connLimit int64, db *database.DB, conf *config.Config, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		chunkStr:         strconv.Itoa(chunk),
		connLimit:        connLimit,
		db:               db,
		conf:             conf,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *Input),
	}
}

//collect sends a group of domains to be analyzed
func (a *analyzer) collect(datum *Input) {
	a.analysisChannel <- datum
}

//close waits for the collector to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

//start kicks off a new analysis thread
func (a *analyzer) start() {
	a.analysisWg.Add(1)
	go func() {

		for datum := range a.analysisChannel {
			// set up writer output
			var output update

			// create query
			query := bson.M{}

			tuples := datum.Tuples.Items()
			if len(tuples) > 5 {
				tuples = tuples[:5]
			}

			// Tally up the bytes and duration from the open connections.
			// We will add these at the top level of the current uconn entry
			// when it's placed in mongo such that we will have an up-to-date
			// total for open connection values each time we parse another
			// set of logs. These current values will overwrite any existing values.
			// The relevant values from the closed connection will be added to the
			// appropriate chunk in a "dat" and those values will effetively be
			// removed from the open connection values that we are tracking.
			for key, connStateEntry := range datum.ConnStateMap {
				if connStateEntry.Open {
					datum.OpenBytes += connStateEntry.Bytes
					datum.OpenDuration += connStateEntry.Duration
					datum.OpenOrigBytes += connStateEntry.OrigBytes

					//Increment the OpenConnectionCount for each open entry that we have
					datum.OpenConnectionCount++

					// Only append unique timestamps to OpenTsList.
					if !int64InSlice(connStateEntry.Ts, datum.OpenTSList) {
						datum.OpenTSList = append(datum.OpenTSList, connStateEntry.Ts)
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

			// if this connection qualifies to be a strobe with the current number
			// of connections in the current datum, don't store bytes and ts.
			// it will not qualify to be downgraded to a beacon until this chunk is
			// outdated and removed. If only importing once - still just a strobe.
			if datum.ConnectionCount >= a.connLimit {
				query["$set"] = bson.M{
					"strobe":                true,
					"cid":                   a.chunk,
					"src_network_name":      datum.Hosts.SrcNetworkName,
					"dst_network_name":      datum.Hosts.DstNetworkName,
					"open":                  connState,
					"open_bytes":            datum.OpenBytes,
					"open_connection_count": datum.OpenConnectionCount,
					"open_conns":            datum.ConnStateMap,
					"open_duration":         datum.OpenDuration,
					"open_orig_bytes":       datum.OpenOrigBytes,
					"open_ts":               datum.OpenTSList,
				}
				query["$push"] = bson.M{
					"dat": bson.M{
						"count":  datum.ConnectionCount,
						"bytes":  []interface{}{},
						"ts":     []interface{}{},
						"tuples": tuples,
						"icerts": datum.InvalidCertFlag,
						"maxdur": datum.MaxDuration,
						"tbytes": datum.TotalBytes,
						"tdur":   datum.TotalDuration,
						"cid":    a.chunk,
					},
				}
			} else {
				query["$set"] = bson.M{
					"cid":                   a.chunk,
					"src_network_name":      datum.Hosts.SrcNetworkName,
					"dst_network_name":      datum.Hosts.DstNetworkName,
					"open":                  connState,
					"open_bytes":            datum.OpenBytes,
					"open_connection_count": datum.OpenConnectionCount,
					"open_conns":            datum.ConnStateMap,
					"open_duration":         datum.OpenDuration,
					"open_orig_bytes":       datum.OpenOrigBytes,
					"open_ts":               datum.OpenTSList,
				}
				query["$push"] = bson.M{
					"dat": bson.M{
						"count":  datum.ConnectionCount,
						"bytes":  datum.OrigBytesList,
						"ts":     datum.TsList,
						"tuples": tuples,
						"icerts": datum.InvalidCertFlag,
						"maxdur": datum.MaxDuration,
						"tbytes": datum.TotalBytes,
						"tdur":   datum.TotalDuration,
						"cid":    a.chunk,
					},
				}
			}

			// assign formatted query to output
			output.uconn.query = query

			output.uconn.selector = datum.Hosts.BSONKey()

			// get maxdur host table update
			// since we are only updating stats for internal ips (as defined by the
			// user in the file), we need to customize the query to update based on
			// which ip in the connection was local.
			if datum.IsLocalSrc {
				output.hostMaxDur = a.hostMaxDurQuery(datum.MaxDuration, datum.Hosts.UniqueSrcIP.Unpair(), datum.Hosts.UniqueDstIP.Unpair())
			} else if datum.IsLocalDst {
				output.hostMaxDur = a.hostMaxDurQuery(datum.MaxDuration, datum.Hosts.UniqueDstIP.Unpair(), datum.Hosts.UniqueSrcIP.Unpair())
			}

			// set to writer channel
			a.analyzedCallback(output)

		}
		a.analysisWg.Done()
	}()
}

func (a *analyzer) hostMaxDurQuery(maxDur float64, localIP data.UniqueIP, externalIP data.UniqueIP) updateInfo {
	ssn := a.db.Session.Copy()
	defer ssn.Close()

	var output updateInfo

	// create query
	query := bson.M{}

	// check if we need to update
	// we do this before the other queries because otherwise if a max dur
	// starts out with a high number which reduces over time, it will keep
	// the incorrect high max for that specific destination.
	var resListExactMatch []interface{}

	maxDurMatchExactQuery := localIP.BSONKey()
	maxDurMatchExactQuery["dat"] = bson.M{
		"$elemMatch": bson.M{
			"mdip":         externalIP.BSONKey(),
			"max_duration": bson.M{"$lte": maxDur},
		},
	}

	_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(maxDurMatchExactQuery).All(&resListExactMatch)

	// if we have exact matches, update to new score and return
	if len(resListExactMatch) > 0 {

		// update chunk number
		query["$set"] = bson.M{
			"dat.$.cid":          a.chunk,
			"dat.$.max_duration": maxDur,
		}

		// create selector for output
		output.query = query

		// using the same find query we created above will allow us to match and
		// update the exact chunk we need to update
		output.selector = maxDurMatchExactQuery

		return output
	}

	// The below is only for cases where the ip is not currently listed as a max dur
	// for a source
	// update max dur
	newFlag := false
	updateFlag := false

	var resListLower []interface{}
	var resListUpper []interface{}

	// this query will find any matching chunk that is reporting a lower
	// max beacon score than the current one we are working with
	maxDurMatchLowerQuery := localIP.BSONKey()
	maxDurMatchLowerQuery["dat"] = bson.M{
		"$elemMatch": bson.M{
			"cid":          a.chunk,
			"max_duration": bson.M{"$lte": maxDur},
		},
	}

	maxDurMatchUpperQuery := localIP.BSONKey()
	maxDurMatchUpperQuery["dat"] = bson.M{
		"$elemMatch": bson.M{
			"cid":          a.chunk,
			"max_duration": bson.M{"$gte": maxDur},
		},
	}

	// find matching lower chunks
	_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(maxDurMatchLowerQuery).All(&resListLower)

	// update if there are lower entries in this chunk
	if len(resListLower) > 0 {
		updateFlag = true
	} else {
		// find matching upper records in this chunk
		_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(maxDurMatchUpperQuery).All(&resListUpper)

		// create a new entry if there are no bigger entries
		if len(resListUpper) <= 0 {
			newFlag = true
		}
	}

	// since we didn't find any changeable lower max duration scores, we will
	// set the condition to push a new entry with the current score listed as the
	// max beacon ONLY if no matching chunks reporting higher max beacon scores
	// are found.

	if newFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"max_duration": maxDur,
				"mdip":         externalIP,
				"cid":          a.chunk,
			}}

		// create selector for output
		output.query = query
		output.selector = localIP.BSONKey()

	} else if updateFlag {
		query["$set"] = bson.M{
			"dat.$.max_duration": maxDur,
			"dat.$.mdip":         externalIP,
			"dat.$.cid":          a.chunk,
		}

		// create selector for output
		output.query = query

		// using the same find query we created above will allow us to match and
		// update the exact chunk we need to update
		output.selector = maxDurMatchLowerQuery
	}

	return output
}

//int64InSlice ...
func int64InSlice(a int64, list []int64) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
