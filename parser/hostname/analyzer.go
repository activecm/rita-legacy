package hostname

import (
	"strings"
	"sync"

	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for exploded dns analysis
	analyzer struct {
		analyzedCallback func(update)   // called on each analyzed result
		closedCallback   func()         // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan hostname  // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for parsing hostnames
func newAnalyzer(analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan hostname),
	}
}

//collect sends a group of domains to be analyzed
func (a *analyzer) collect(data hostname) {
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

			// in some of these strings, the empty space will get counted as a domain,
			// this was an issue in the old version of exploded dns and caused inaccuracies
			if (data.name == "") || (strings.HasSuffix(data.name, "in-addr.arpa")) {
				continue
			}

			// create query
			output.query = bson.M{
				"$setOnInsert": bson.M{"host": data.name},
				"$addToSet":    bson.M{"ips": bson.M{"$each": data.answers}},
			}

			// create selector for output
			output.selector = bson.M{"host": data.name}

			// set to writer channel
			a.analyzedCallback(output)

		}
		a.analysisWg.Done()
	}()
}
