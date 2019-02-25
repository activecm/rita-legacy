package useragent

import (
	"strconv"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for exploded dns analysis
	analyzer struct {
		chunk            int            //current chunk (0 if not on rolling analysis)
		chunkStr         string         //current chunk (0 if not on rolling analysis)
		db               *database.DB   // provides access to MongoDB
		conf             *config.Config // contains details needed to access MongoDB
		analyzedCallback func(update)   // called on each analyzed result
		closedCallback   func()         // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Input    // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for parsing hostnames
func newAnalyzer(chunk int, db *database.DB, conf *config.Config, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		chunkStr:         strconv.Itoa(chunk),
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
			query := bson.M{
				"$push": bson.M{
					"dat": bson.M{
						"seen":     data.Seen,
						"orig_ips": data.OrigIps,
						"hosts":    data.Requests,
						"cid":      a.chunk,
					},
				},
				"$set":         bson.M{"cid": a.chunk},
				"$setOnInsert": bson.M{"ja3": data.JA3},
			}

			output.query = query

			output.collection = a.conf.T.UserAgent.UserAgentTable
			// create selector for output
			output.selector = bson.M{"user_agent": data.name}

			// set to writer channel
			a.analyzedCallback(output)

			// this is for flagging rarely used j3 and useragent hosts
			if len(data.OrigIps) < 5 {
				maxLeft := 5 - len(data.OrigIps)

				query := []bson.M{
					bson.M{"$match": bson.M{"user_agent": data.name}},
					bson.M{"$project": bson.M{"ips": "$dat.orig_ips", "user_agent": 1}},
					bson.M{"$unwind": "$ips"},
					bson.M{"$unwind": "$ips"}, // not an error, needs to be done twice
					bson.M{"$group": bson.M{
						"_id": "$user_agent",
						"ips": bson.M{"$addToSet": "$ips"},
					}},
					bson.M{"$project": bson.M{
						"count": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$ips", []interface{}{}}}},
						"ips":   "$ips",
					}},
					bson.M{"$match": bson.M{"count": bson.M{"$lte": maxLeft}}},
				}

				var resList struct {
					ID    string   `bson:"_id"`
					IPS   []string `bson:"ips"`
					Count int      `bson:"count"`
				}

				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.UserAgent.UserAgentTable).Pipe(query).One(&resList)

				for _, entry := range resList.IPS {

					newRecordFlag := false

					type hostRes struct {
						CID int `bson:"cid"`
					}

					var res2 []hostRes

					_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(bson.M{"ip": entry, "dat.rsig": data.name}).All(&res2)

					if !(len(res2) > 0) {
						newRecordFlag = true
						// fmt.Println("host no results", res2, data.Host)
					} else {

						if res2[0].CID != a.chunk {
							// fmt.Println("host existing", a.chunk, res2, data.Host)
							newRecordFlag = true
						}
					}

					output := hostQuery(a.chunk, data.name, entry, newRecordFlag)
					output.collection = a.conf.T.Structure.HostTable

					// set to writer channel
					a.analyzedCallback(output)

				}
			}

		}

		a.analysisWg.Done()
	}()
}

//hostQuery ...
func hostQuery(chunk int, useragentStr string, ip string, newFlag bool) update {
	var output update

	// create query
	query := bson.M{}

	if newFlag {
		query["$push"] = bson.M{
			"dat": bson.M{
				"rsig":  useragentStr,
				"rsigc": 1,
				"cid":   chunk,
			}}

		// create selector for output ,
		output.query = query
		output.selector = bson.M{"ip": ip}

	} else {

		query["$set"] = bson.M{
			"dat.$.rsigc": 1,
			"dat.$.chunk": chunk,
		}

		// create selector for output
		output.query = query
		output.selector = bson.M{"ip": ip, "dat.cid": chunk}
	}

	return output
}
