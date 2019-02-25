package uconn

import (
	"strconv"
	"sync"

	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for exploded dns analysis
	analyzer struct {
		chunk            int            //current chunk (0 if not on rolling analysis)
		chunkStr         string         //current chunk (0 if not on rolling analysis)
		connLimit        int64          // limit for strobe classification
		analyzedCallback func(update)   // called on each analyzed result
		closedCallback   func()         // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Pair     // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for parsing hostnames
func newAnalyzer(chunk int, connLimit int64, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		chunkStr:         strconv.Itoa(chunk),
		connLimit:        connLimit,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *Pair),
	}
}

//collect sends a group of domains to be analyzed
func (a *analyzer) collect(data *Pair) {
	a.analysisChannel <- data
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

		for data := range a.analysisChannel {
			// set up writer output
			var output update

			// create query
			query := bson.M{
				"$setOnInsert": bson.M{
					"local_src": data.IsLocalSrc,
					"local_dst": data.IsLocalDst,
				},
			}

			if len(data.Tuples) > 5 {
				data.Tuples = data.Tuples[:5]
			}

			// if this connection qualifies to be a strobe with the current number
			// of connections in the currently parsing in data, don't store bytes and ts.
			// it will not qualify to be downgraded to a beacon until this chunk is
			// outdated and removed. If only importing once - still just a strobe.
			if data.ConnectionCount >= a.connLimit {
				query["$set"] = bson.M{"strobe": true, "cid": a.chunk}
				query["$push"] = bson.M{
					"dat": bson.M{
						"count":  data.ConnectionCount,
						"bytes":  []interface{}{},
						"ts":     []interface{}{},
						"tuples": data.Tuples,
						"maxdur": data.MaxDuration,
						"tbytes": data.TotalBytes,
						"tdur":   data.TotalDuration,
						"cid":    a.chunk,
					},
				}
			} else {
				query["$push"] = bson.M{
					"dat": bson.M{
						"count":  data.ConnectionCount,
						"bytes":  data.OrigBytesList,
						"ts":     data.TsList,
						"tuples": data.Tuples,
						"maxdur": data.MaxDuration,
						"tbytes": data.TotalBytes,
						"tdur":   data.TotalDuration,
						"cid":    a.chunk,
					},
				}
				query["$set"] = bson.M{"cid": a.chunk}
			}

			// assign formatted query to output
			output.query = query

			// create selector for output
			output.selector = bson.M{"src": data.Src, "dst": data.Dst}

			// set to writer channel
			a.analyzedCallback(output)

		}
		a.analysisWg.Done()
	}()
}
