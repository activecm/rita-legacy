package useragent

import (
	"strconv"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/data"
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
func (a *analyzer) collect(datum *Input) {
	a.analysisChannel <- datum
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

		for datum := range a.analysisChannel {

			if len(datum.OrigIps) > 10 {
				datum.OrigIps = datum.OrigIps[:10]
			}

			if len(datum.Requests) > 10 {
				datum.Requests = datum.Requests[:10]
			}

			// create query
			query := bson.M{
				"$push": bson.M{
					"dat": bson.M{
						"seen":     datum.Seen,
						"orig_ips": datum.OrigIps,
						"hosts":    datum.Requests,
						"cid":      a.chunk,
					},
				},
				"$set":         bson.M{"cid": a.chunk},
				"$setOnInsert": bson.M{"ja3": datum.JA3},
			}

			// create selector for output
			selector := bson.M{"user_agent": datum.Name}

			// send to writer channel
			a.analyzedCallback(update{useragent: upsertInfo{
				query:    query,
				selector: selector,
			}})

			// this is for flagging rarely used j3 and useragent hosts
			if len(datum.OrigIps) < 5 {
				maxLeft := 5 - len(datum.OrigIps)

				query := []bson.M{
					{"$match": bson.M{"user_agent": datum.Name}},
					{"$project": bson.M{
						"dat.orig_ips.network_name": 0, // drop network_name before UniqueIP comparisons
					}},
					{"$project": bson.M{"ips": "$dat.orig_ips", "user_agent": 1}},
					{"$unwind": "$ips"},
					{"$unwind": "$ips"}, // not an error, needs to be done twice
					{"$group": bson.M{
						"_id": "$user_agent",
						"ips": bson.M{"$addToSet": "$ips"},
					}},
					{"$project": bson.M{
						"count": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$ips", []interface{}{}}}},
						"ips":   "$ips",
					}},
					{"$match": bson.M{"count": bson.M{"$lte": maxLeft}}},
				}

				var rareSigList struct {
					OrigIps []data.UniqueIP `bson:"ips"`
				}

				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.UserAgent.UserAgentTable).Pipe(query).AllowDiskUse().One(&rareSigList)

				for _, rareSigIP := range rareSigList.OrigIps {

					newRecordFlag := false // have we created a rare signature entry for this host in this chunk yet?

					type hostEntry struct {
						CID int `bson:"cid"`
					}

					var hostEntries []hostEntry

					entryHostQuery := rareSigIP.BSONKey()
					entryHostQuery["dat.rsig"] = datum.Name
					//TODO: Consider adding the chunk to the query instead of checking it after the query

					_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(entryHostQuery).All(&hostEntries)

					//TODO: I think there might be a bug here. From what I can tell, if we set the new record flag
					// when there is a rare signature record from a previous chunk, then we will end up pushing a duplicate
					// record for each chunk. We should investigate removing this chunk check here. -LL
					if len(hostEntries) <= 0 || hostEntries[0].CID != a.chunk {
						newRecordFlag = true
					}

					updateInfo := hostQuery(a.chunk, datum.Name, rareSigIP, newRecordFlag)

					// set to writer channel
					a.analyzedCallback(update{host: updateInfo})

				}
			}

		}

		a.analysisWg.Done()
	}()
}

//hostQuery ...
func hostQuery(chunk int, useragentStr string, ip data.UniqueIP, newFlag bool) updateWithArrayFiltersInfo {
	var output updateWithArrayFiltersInfo

	// create query
	query := bson.M{}

	if newFlag {
		query["$push"] = bson.M{
			"dat": bson.M{
				"rsig":  useragentStr,
				"rsigc": 1,
				"cid":   chunk,
			},
		}
		output.query = query

		// create selector for output ,
		output.selector = ip.BSONKey()

	} else {

		query["$set"] = bson.M{
			"dat.$[t].rsigc": 1,
			"dat.$[t].cid":   chunk,
		}
		output.query = query

		output.arrayFilters = []bson.M{{
			"t.rsig": useragentStr,
		}}

		// create selector for output
		// we don't add cid to the selector query because if the useragent string is
		// already listed, we just want to update it to the most recent chunk instead
		// of adding more
		output.selector = ip.BSONKey()
		output.selector["dat.rsig"] = useragentStr
	}

	return output
}
