package blacklist

import (
	"fmt"
	"sync"

	"github.com/activecm/rita/database"
	"github.com/activecm/rita/datatypes/blacklist"
	"github.com/globalsign/mgo/bson"
)

type (
	// analyzer implements the bulk of beaconing analysis, creating the scores
	// for a given set of timestamps and data sizes
	analyzer struct {
		db               *database.DB                    // provides access to MongoDB
		analyzedCallback func(*blacklist.AnalysisOutput) // called on each analyzed result
		closedCallback   func()                          // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *blacklist.AnalysisInput   // holds unanalyzed data
		analysisWg       sync.WaitGroup                  // wait for analysis to finish
	}
)

// newAnalyzer creates a new analyzer for computing beaconing scores.
func newAnalyzer(db *database.DB, analyzedCallback func(*blacklist.AnalysisOutput), closedCallback func()) *analyzer {
	fmt.Println("-- new Analyzer --")
	return &analyzer{
		db:               db,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *blacklist.AnalysisInput),
	}
}

// analyze sends a group of timestamps and data sizes in for analysis.
// Note: this function may block
func (a *analyzer) analyze(data *blacklist.AnalysisInput) {
	a.analysisChannel <- data
}

// close waits for the analysis threads to finish
func (a *analyzer) close() {
	fmt.Println("-- close Analyzer --")
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

// start kicks off a new analysis thread
func (a *analyzer) start() {
	fmt.Println("-- start Analyzer --")
	a.analysisWg.Add(1)
	go func() {
		ssn := a.db.Session.Copy()
		defer ssn.Close()

		for data := range a.analysisChannel {

			var resList []blacklist.IPResult
			_ = ssn.DB("rita-bl").C("ip").Find(bson.M{"index": data.IP}).All(&resList)

			//if the ip address has blacklist results
			if len(resList) > 0 {

				// initialize the output structure
				output := &blacklist.AnalysisOutput{IP: data.IP}

				// Get all blacklists result was found on
				for _, entry := range resList {
					// fmt.Println(entry.List)
					output.Lists = append(output.Lists, entry.List)
				}
				// 			err := fillBlacklistedIP(
				// 				&blIP,
				// 				res.DB.GetSelectedDB(),
				// 				res.Config.T.Structure.UniqueConnTable,
				// 				res.Config.T.Structure.HostTable,
				// 				ssn,
				// 				source,
				// 			)
				// 			if err != nil {
				// 				res.Log.WithFields(log.Fields{
				// 					"err": err.Error(),
				// 					"ip":  ipAddr,
				// 					"db":  res.DB.GetSelectedDB(),
				// 				}).Error("could not aggregate info on blacklisted IP")
				// 				continue
				// }
				// 			outputCollection.Insert(&blIP)
				//
				fmt.Println(resList)
				a.analyzedCallback(output)
			} else {
				continue
			}

			// output := &blacklist.AnalysisOutput{
			// 	IP: data.IP,
			// }

		}
		a.analysisWg.Done()
	}()
}
