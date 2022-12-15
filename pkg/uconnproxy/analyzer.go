package uconnproxy

import (
	"strconv"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for proxy beacon analysis
	analyzer struct {
		chunk            int                        //current chunk (0 if not on rolling analysis)
		chunkStr         string                     //current chunk (0 if not on rolling analysis)
		connLimit        int64                      // limit for strobe classification
		db               *database.DB               // provides access to MongoDB
		conf             *config.Config             // contains details needed to access MongoDB
		analyzedCallback func(database.BulkChanges) // called on each analyzed result
		closedCallback   func()                     // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Input                // holds unanalyzed data
		analysisWg       sync.WaitGroup             // wait for analysis to finish
	}
)

// newAnalyzer creates a new collector for parsing uconnproxy
func newAnalyzer(chunk int, connLimit int64, db *database.DB, conf *config.Config, analyzedCallback func(database.BulkChanges), closedCallback func()) *analyzer {
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

// collect sends a group of uconnproxy data to be analyzed
func (a *analyzer) collect(datum *Input) {
	a.analysisChannel <- datum
}

// close waits for the collector to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

// start kicks off a new analysis thread
func (a *analyzer) start() {
	a.analysisWg.Add(1)
	go func() {

		for datum := range a.analysisChannel {

			mainUpdate := mainQuery(datum, a.connLimit, a.chunk)

			a.analyzedCallback(database.BulkChanges{
				a.conf.T.Structure.UniqueConnProxyTable: []database.BulkChange{{
					Selector: datum.Hosts.BSONKey(),
					Update:   mainUpdate,
					Upsert:   true,
				}},
			})
		}
		a.analysisWg.Done()
	}()
}

// mainQuery records the bulk of the information about communications between two hosts
// over an HTTP proxy
func mainQuery(datum *Input, strobeLimit int64, chunk int) bson.M {

	// if this connection qualifies to be a strobe with the current number
	// of connections in the current datum, don't store ts.
	// it will not qualify to be downgraded to a proxy beacon until this chunk is
	// outdated and removed. If only importing once - still just a strobe.
	ts := datum.TsList

	isStrobe := datum.ConnectionCount >= strobeLimit
	if isStrobe {
		ts = []int64{}
	}

	return bson.M{
		"$set": bson.M{
			"strobeFQDN":       isStrobe,
			"cid":              chunk,
			"src_network_name": datum.Hosts.SrcNetworkName,
			"proxy":            datum.Proxy,
		},
		"$push": bson.M{
			"dat": bson.M{
				"$each": []bson.M{{
					"count": datum.ConnectionCount,
					"ts":    ts,
					"cid":   chunk,
				}},
			},
		},
	}
}
