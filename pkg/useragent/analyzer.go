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

			if len(data.OrigIps) > 10 {
				data.OrigIps = data.OrigIps[:10]
			}

			if len(data.Requests) > 10 {
				data.Requests = data.Requests[:10]
			}
			// create query
			output.query = bson.M{
				"$push": bson.M{
					"dat": bson.M{
						"seen":     data.Seen,
						"orig_ips": data.OrigIps,
						"hosts":    data.Requests,
					},
				},
				"$inc": bson.M{"times_used": data.Seen},
			}

			// create selector for output
			output.selector = bson.M{"user_agent": data.name}

			// set to writer channel
			a.analyzedCallback(output)

		}

		a.analysisWg.Done()
	}()
}
