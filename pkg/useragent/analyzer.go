package useragent

import (
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
		analysisChannel  chan *Input    // holds unanalyzed data
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
		analysisChannel:  make(chan *Input),
	}
}

//collect sends a group of domains to be analyzed
func (a *analyzer) collect(data *Input) {
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
			// set up writer output
			var output update

			var res useragent

			_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.UserAgent.UserAgentTable).Find(bson.M{"user_agent": data.name}).Limit(1).One(&res)

			// Check for errors and parse results
			if len(res.ips) < a.conf.S.Hostname.IPListLimit {

				// get max we can still add to the array
				max := a.conf.S.Hostname.IPListLimit - len(res.ips)

				// if we're under max (most cases), continue
				// otherwise we'll need to parse the correct size.
				if len(data.Ips) >= max {
					removeDuplicates(data.Ips, res.ips, max)
				}

				// create query
				output.query = bson.M{
					"$addToSet": bson.M{"ips": bson.M{"$each": data.Ips}},
					"$inc":      bson.M{"times_used": data.Seen},
				}

				// create selector for output
				output.selector = bson.M{"user_agent": data.name}

				// set to writer channel
				a.analyzedCallback(output)
			}

		}

		a.analysisWg.Done()
	}()
}

func removeDuplicates(s1 []string, s2 []string, max int) []string {
	// i know... but it will happen only for very extreme cases. and on only 2 hours of data.
	// feel free to tear it apart for something better.
	var parsed []string
	for _, entry1 := range s1 {
		found := false
		for _, entry2 := range s2 {
			if entry1 == entry2 {
				found = true
				break
			}
		}
		if !found {
			parsed = append(parsed, entry1)
		}

		if len(parsed) >= max {
			break
		}
	}
	return parsed
}
