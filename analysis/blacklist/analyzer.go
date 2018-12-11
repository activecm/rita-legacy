package blacklist

import (
	"fmt"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/datatypes/blacklist"
	"github.com/globalsign/mgo/bson"
)

type (
	// analyzer implements the bulk of beaconing analysis, creating the scores
	// for a given set of timestamps and data sizes
	analyzer struct {
		source           bool
		db               *database.DB                    // provides access to MongoDB
		conf             *config.Config                  // contains details needed to access MongoDB
		analyzedCallback func(*blacklist.AnalysisOutput) // called on each analyzed result
		closedCallback   func()                          // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *blacklist.AnalysisInput   // holds unanalyzed data
		analysisWg       sync.WaitGroup                  // wait for analysis to finish
	}
)

// newAnalyzer creates a new analyzer for computing beaconing scores.
func newAnalyzer(source bool, db *database.DB, conf *config.Config, analyzedCallback func(*blacklist.AnalysisOutput), closedCallback func()) *analyzer {
	fmt.Println("-- new Analyzer --")
	return &analyzer{
		source:           source,
		db:               db,
		conf:             conf,
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
				output := &blacklist.AnalysisOutput{
					IP:                data.IP,
					Connections:       data.Connections,
					UniqueConnections: data.UniqueConnections,
					TotalBytes:        data.TotalBytes,
					AverageBytes:      data.AverageBytes,
					Targets:           data.Targets,
				}

				// Get all blacklists result was found on
				for _, entry := range resList {
					// fmt.Println(entry.List)
					output.Lists = append(output.Lists, entry.List)
				}

				if a.source {
					for _, src := range data.Targets {
						// If the blacklisted IP initiated the connection, then bl_in_count
						// holds the number of unique blacklisted IPs connected to the given
						// host.
						// bl_sum_avg_bytes adds the average number of bytes over all
						// individual connections between these two systems. This is an
						// indication of how much data was transferred overall but not take
						// into account the number of connections.
						// bl_total_bytes adds the total number of bytes sent over all
						// individual connections between the two systems.
						ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Update(
							bson.M{"ip": src},
							bson.D{
								{"$inc", bson.M{"bl_in_count": 1}},
								{"$set", bson.M{"bl_sum_avg_bytes": data.AverageBytes}},
								{"$set", bson.M{"bl_total_bytes": data.TotalBytes}},
							})
					}

				} else {
					for _, dst := range data.Targets {
						// If the internal system initiated the connection, then bl_out_count
						// holds the number of unique blacklisted IPs the given host contacted.
						// bl_sum_avg_bytes and bl_total_bytes are the same as above.
						ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Update(
							bson.M{"ip": dst},
							bson.D{
								{"$inc", bson.M{"bl_out_count": 1}},
								{"$set", bson.M{"bl_sum_avg_bytes": data.AverageBytes}},
								{"$set", bson.M{"bl_total_bytes": data.TotalBytes}},
							})

					}
				}
				a.analyzedCallback(output)
			} else {
				continue
			}

		}
		a.analysisWg.Done()
	}()
}
