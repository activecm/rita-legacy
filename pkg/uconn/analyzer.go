package uconn

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer records data regarding the connections between pairs of hosts
	analyzer struct {
		chunk            int            //current chunk (0 if not on rolling analysis)
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

			mainUpdate := mainQuery(datum, a.connLimit, a.chunk)
			openConnsUpdate := openConnectionsQuery(datum)

			totalUpdate := database.MergeBSONMaps(mainUpdate, openConnsUpdate)

			a.analyzedCallback(update{
				selector: datum.Hosts.BSONKey(),
				query:    totalUpdate,
			})

		}
		a.analysisWg.Done()
	}()
}

//mainQuery records the bulk of the information about the communications between two hosts
func mainQuery(datum *Input, strobeLimit int64, chunk int) bson.M {

	// Truncate the protocol/ port tuples we store in the database
	tuples := datum.Tuples.Items()
	if len(tuples) > 5 {
		tuples = tuples[:5]
	}

	ts := datum.TsList
	bytes := datum.OrigBytesList

	// if this connection qualifies to be a strobe with the current number
	// of connections in the current datum, don't store bytes and ts.
	// it will not qualify to be downgraded to a beacon until this chunk is
	// outdated and removed. If only importing once - still just a strobe.
	isStrobe := datum.ConnectionCount >= strobeLimit
	if isStrobe {
		ts = []int64{}
		bytes = []int64{}
	}

	return bson.M{
		"$set": bson.M{
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

//openConnectionsQuery records information about connections that are still open between two hosts
func openConnectionsQuery(datum *Input) bson.M {
	var bytes int64
	var duration float64
	var origBytes int64
	var connections int64
	tsList := make([]int64, 0)

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
			bytes += connStateEntry.Bytes
			duration += connStateEntry.Duration
			origBytes += connStateEntry.OrigBytes

			connections++

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

	return bson.M{
		"$set": bson.M{
			"open":                  connState,
			"open_bytes":            bytes,
			"open_connection_count": connections,
			"open_conns":            datum.ConnStateMap,
			"open_duration":         duration,
			"open_orig_bytes":       origBytes,
			"open_ts":               tsList,
		},
	}
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
