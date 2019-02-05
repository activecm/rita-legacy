package explodeddns

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
		analysisChannel  chan domain    // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for parsing subdomains
func newAnalyzer(analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan domain),
	}
}

//collect sends a group of domains to be analyzed
func (a *analyzer) collect(data domain) {
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

			// split name on periods
			split := strings.Split(data.name, ".")

			// we will not count the very last item, because it will be either all or
			// a part of the tlds. This means that something like ".co.uk" will still
			// not be fully excluded, but it will greatly reduce the complexity for the
			// most common tlds
			max := len(split) - 1

			for i := 1; i <= max; i++ {
				// parse domain which will be the part we are on until the end of the string
				entry := strings.Join(split[max-i:], ".")

				// in some of these strings, the empty space will get counted as a domain,
				// this was an issue in the old version of exploded dns and caused inaccuracies
				if (entry == "") || (entry == "in-addr.arpa") {
					break
				}

				// set up writer output
				var output update

				// get subdomains
				subdomains := strings.Join(split[0:max-i], ".")

				// create qquery for output depending on whether the domain has subdomains
				if subdomains != "" {
					output.query = bson.M{
						"$setOnInsert": bson.M{"domain": entry},
						"$addToSet":    bson.M{"subdomains": subdomains},
						"$inc":         bson.M{"visited": data.count},
					}
				} else {
					output.query = bson.M{"$setOnInsert": bson.M{"domain": entry}, "$inc": bson.M{"visited": data.count}}
				}

				// create selector for output
				output.selector = bson.M{"domain": entry}

				// set to writer channel
				a.analyzedCallback(output)

			}

		}
		a.analysisWg.Done()
	}()
}
