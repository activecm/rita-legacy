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
			output.userAgent.query = bson.M{
				"$push": bson.M{
					"dat": bson.M{
						"seen":     data.Seen,
						"orig_ips": data.OrigIps,
						"hosts":    data.Requests,
					},
				},
				"$inc": bson.M{"times_used": data.Seen},
			}

			if len(data.OrigIps) < 5 {
				maxLeft := 5 - len(data.OrigIps)

				query := []bson.M{
					bson.M{"$match": bson.M{"user_agent": data.name}},
					bson.M{"$unwind": "$dat"},
					bson.M{"$unwind": "$dat.orig_ips"},
					bson.M{"$group": bson.M{
						"_id": "$_.id",
						"ips": bson.M{"$addToSet": "$dat.orig_ips"},
					}},
					bson.M{"$project": bson.M{
						"count": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$ips", []interface{}{}}}},
						"ips":   "$ips",
					}},
					bson.M{"$match": bson.M{"count": bson.M{"$lt": maxLeft}}},
				}

				var resList []struct {
					ips   []string `bson:"ips"`
					count int      `bson:"count"`
				}

				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.UserAgent.UserAgentTable).Pipe(query).All(&resList)

				if len(resList) > 0 {
					output.host.query = bson.M{
						"$push": bson.M{
							//TODO still need to find a query to get blacklisted, and ipv4_binary
							//  accurately added to the host database
							"blacklisted": "true",
							"dat": bson.M{
								"$inc": bson.M{"count_dst": resList[0].count},
							},
							"ipv4_binary": "3518000993",
							"local":       "true",
						},
					}
					output.host.selector = bson.M{"ip": resList[0].ips[0]}
					// flag both the ips in the returned result and the current ones.
					// you will need to add another output here. since its going to a different collection than useragent,
					// look at the beacons pkg for how i'm doing that. You'll need an output struct with other table outputs.
				}
			}

			// create selector for output
			output.userAgent.selector = bson.M{"user_agent": data.name}

			// set to writer channel
			a.analyzedCallback(output)

		}

		a.analysisWg.Done()
	}()
}
