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

			var res hostname

			_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.DNS.HostnamesTable).Find(bson.M{"host": data.host}).Limit(1).One(&res)

			// get max we can still add to the array
			max := a.conf.S.Hostname.IPListLimit - len(res.ips)

			// if we're under max (most cases), continue
			// otherwise we'll need to parse the correct size (rare)
			if len(data.ips) >= max {
				removeDuplicates(data.ips, res.ips, max)
			}

			// set blacklisted Flag
			blacklistFlag := false

			// check ip against blacklist
			var resList []ritaBLResult
			_ = ssn.DB(a.conf.S.Blacklisted.BlacklistDatabase).C("hostname").Find(bson.M{"index": data.host}).All(&resList)

			// variable holding blacklist stats that's only assigned values if there is a blacklisted result
			var uconnStats uconnRes

			// check if hostname is blacklisted
			if len(resList) > 0 {
				// set blacklist flag to true for hostname
				blacklistFlag = true

				// get non-overflowed list of ips to use in stats query
				ipList := res.ips
				if len(ipList) < a.conf.S.Hostname.IPListLimit {
					ipList = append(ipList, data.ips...)
				}

				// build query
				uconnsQuery := getBlacklistsStatsQuery(ipList)

				// get stats
				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.UniqueConnTable).Pipe(uconnsQuery).One(&uconnStats)

			}

			// if current array length on the server is under the limit, we update the record
			if len(res.ips) < a.conf.S.Hostname.IPListLimit {
				// set up writer output
				var output update

				// create query
				if blacklistFlag {

					output.query = bson.M{
						"$addToSet": bson.M{"ips": bson.M{"$each": data.ips}},
						"$set": bson.M{
							"blacklisted": true,
							"conn_count":  uconnStats.Connections,
							"uconn_count": uconnStats.UniqueConnections,
							"total_bytes": uconnStats.TotalBytes,
						},
					}
				} else {
					output.query = bson.M{
						"$addToSet": bson.M{"ips": bson.M{"$each": data.ips}},
					}
				}

				// create selector for output
				output.selector = bson.M{"host": data.host}

				// set to writer channel
				a.analyzedCallback(output)

			} else if blacklistFlag { // otherwise, we only update the record if its blacklisted
				// set up writer output
				var output update

				// create query
				output.query = bson.M{
					"$set": bson.M{
						"blacklisted": true,
						"conn_count":  uconnStats.Connections,
						"uconn_count": uconnStats.UniqueConnections,
						"total_bytes": uconnStats.TotalBytes,
					},
				}

				// create selector for output
				output.selector = bson.M{"host": data.host}
				// set to writer channel
				a.analyzedCallback(output)
			}

		}

		a.analysisWg.Done()
	}()
}

func removeDuplicates(s1 []string, s2 []string, max int) []string {
	// i know... but it will happen very rarely. and on only 2 hours of data.
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

//getBlacklistsStats will only run if a hostname is determined to be a blacklisted hostname
func getBlacklistsStatsQuery(hosts []string) []bson.M {
	//nolint: vet
	return []bson.M{
		bson.M{"$match": bson.M{"dst": bson.M{"$in": hosts}}},
		bson.M{"$group": bson.M{
			"_id":         0,
			"total_bytes": bson.M{"$sum": "$total_bytes"},
			"conn_count":  bson.M{"$sum": "$connection_count"},
			"uconn_count": bson.M{"$sum": 1},
		}},
		bson.M{"$project": bson.M{
			"_id":         0,
			"total_bytes": 1,
			"conn_count":  1,
			"uconn_count": 1,
		}},
	}
}
