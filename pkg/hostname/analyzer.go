package hostname

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
		analysisChannel  chan hostname  // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for parsing hostnames
func newAnalyzer(db *database.DB, conf *config.Config, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		db:               db,
		conf:             conf,
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
		ssn := a.db.Session.Copy()
		defer ssn.Close()

		for data := range a.analysisChannel {

			// in some of these strings, the empty space will get counted as a domain,
			// this was an issue in the old version of exploded dns and caused inaccuracies
			if (data.host == "") || (strings.HasSuffix(data.host, "in-addr.arpa")) {
				continue
			}

			// set blacklisted Flag
			blacklistFlag := false

			// check ip against blacklist
			var resList []ritaBLResult
			_ = ssn.DB(a.conf.S.Blacklisted.BlacklistDatabase).C("hostname").Find(bson.M{"index": data.host}).All(&resList)

			// check if hostname is blacklisted
			if len(resList) > 0 {
				blacklistFlag = true
			}

			// set up writer output
			var output update

			// create query
			if blacklistFlag {
				// flag as blacklisted if blacklisted
				output.query = bson.M{
					"$push": bson.M{
						"dat": bson.M{"ips": data.ips},
					},
					"$set": bson.M{
						"blacklisted": true,
					},
				}
			} else {
				output.query = bson.M{
					"$push": bson.M{
						"dat": bson.M{"ips": data.ips},
					},
				}
			}

			// create selector for output
			output.selector = bson.M{"host": data.host}

			// set to writer channel
			a.analyzedCallback(output)

		}

		a.analysisWg.Done()
	}()
}
