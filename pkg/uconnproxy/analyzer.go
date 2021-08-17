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

//newAnalyzer creates a new collector for parsing uconnproxy
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

//collect sends a group of uconnproxy data to be analyzed
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

			// if this connection qualifies to be a strobe with the current number
			// of connections in the current datum, don't store bytes and ts.
			// it will not qualify to be downgraded to a proxy beacon until this chunk is
			// outdated and removed. If only importing once - still just a strobe.
			if datum.ConnectionCount >= a.connLimit {
				query["$set"] = bson.M{
					"strobe":           true,
					"cid":              a.chunk,
					"src_network_name": datum.Hosts.SrcNetworkName,
					"proxy_ip":         datum.ProxyIP,
				}
				query["$push"] = bson.M{
					"dat": bson.M{
						"count": datum.ConnectionCount,
						"bytes": []interface{}{},
						"ts":    []interface{}{},
						"cid":   a.chunk,
					},
				}
			} else {
				query["$set"] = bson.M{
					"cid":              a.chunk,
					"src_network_name": datum.Hosts.SrcNetworkName,
					"proxy_ip":         datum.ProxyIP,
				}
				query["$push"] = bson.M{
					"dat": bson.M{
						"count": datum.ConnectionCount,
						"ts":    datum.TsList,
						"cid":   a.chunk,
					},
				}
			}

			// assign formatted query to output
			output.uconnProxy.query = query

			output.uconnProxy.selector = datum.Hosts.BSONKey()

			// set to writer channel
			a.analyzedCallback(output)
		}
		a.analysisWg.Done()
	}()
}
