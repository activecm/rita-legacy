package uconn

import (
	"sync"

	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for exploded dns analysis
	analyzer struct {
		analyzedCallback func(update)   // called on each analyzed result
		closedCallback   func()         // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Pair     // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for parsing hostnames
func newAnalyzer(analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
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
			output.query = bson.M{
				"$setOnInsert": bson.M{
					"src":       data.Src,
					"dst":       data.Dst,
					"local_src": data.IsLocalSrc,
					"local_dst": data.IsLocalDst,
				},
				"$addToSet": bson.M{"ts_list": bson.M{"$each": data.TsList}},
				"$inc": bson.M{
					"connection_count": data.ConnectionCount,
					"total_duration":   data.TotalDuration,
					"total_bytes":      data.TotalBytes,
				},
				"$max":  bson.M{"max_duration": data.MaxDuration},
				"$push": bson.M{"orig_bytes_list": bson.M{"$each": data.OrigBytesList}},
			}
			//
			// // create selector for output
			output.selector = bson.M{"src": data.Src, "dst": data.Dst}

			// set to writer channel
			a.analyzedCallback(output)

		}
		a.analysisWg.Done()
	}()
}
