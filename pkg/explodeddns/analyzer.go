package explodeddns

import (
	"strings"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for exploded dns analysis
	analyzer struct {
		db               *database.DB   // provides access to MongoDB
		conf             *config.Config // contains details needed to access MongoDB
		analyzedCallback func(update)   // called on each analyzed result
		closedCallback   func()         // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan domain    // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for parsing subdomains
func newAnalyzer(db *database.DB, conf *config.Config, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		db:               db,
		conf:             conf,
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
		ssn := a.db.Session.Copy()
		defer ssn.Close()
		for data := range a.analysisChannel {

			// check if this query string has already been parsed to add to the subdomain count by checking
			// if the whole string is already in the hostname table.
			var res []struct {
				Host string `bson:"host"`
			}

			_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.DNS.HostnamesTable).Find(bson.M{"host": data.name}).All(&res)

			// flag to keep track of whether we need to increment the subs count
			alreadyCountedSubsFlag := false

			// if its already in the hostnames table, we only need to update the visited count
			if len(res) > 0 {
				alreadyCountedSubsFlag = true
			}

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

				var res2 []interface{}

				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.DNS.ExplodedDNSTable).Find(bson.M{"domain": entry}).All(&res2)

				if !(len(res2) > 0) {
					// set up writer output
					var output update

					output.query = bson.M{
						"$push": bson.M{"dat": bson.M{"visited": data.count}},
						"$inc":  bson.M{"subdomain_count": 1},
					}

					// create selector for output
					output.selector = bson.M{"domain": entry}

					// set to writer channel
					a.analyzedCallback(output)
				} else {
					// set up writer output
					var output update

					if alreadyCountedSubsFlag {
						output.query = bson.M{ // will be updated in future with chunk number
							"$inc": bson.M{"dat.0.visited": data.count},
						}
					} else {
						output.query = bson.M{ // will be updated in future with chunk number
							"$inc": bson.M{"subdomain_count": 1, "dat.0.visited": data.count},
						}
					}

					// create selector for output
					output.selector = bson.M{"domain": entry}

					// set to writer channel
					a.analyzedCallback(output)
				}

			}

		}
		a.analysisWg.Done()
	}()
}
