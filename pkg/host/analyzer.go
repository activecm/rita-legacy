package host

import (
	"encoding/binary"
	"net"
	"strings"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for host analysis
	analyzer struct {
		db               *database.DB     // provides access to MongoDB
		conf             *config.Config   // contains details needed to access MongoDB
		analyzedCallback func(update)     // called on each analyzed result
		closedCallback   func()           // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *uconn.Pair // holds unanalyzed data
		analysisWg       sync.WaitGroup   // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for gathering data
func newAnalyzer(db *database.DB, conf *config.Config, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		db:               db,
		conf:             conf,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *uconn.Pair),
	}
}

//collect sends a chunk of data to be analyzed
func (a *analyzer) collect(data *uconn.Pair) {
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
			// blacklisted flags
			blacklistedSrc := false
			blacklistedDst := false

			// variable holding blacklist stats that's only assigned values if there is a blacklisted result
			var uconnStatsSrc uconnRes
			var uconnStatsDst uconnRes

			// check if blacklisted source
			var resList []ritaBLResult
			_ = ssn.DB(a.conf.S.Blacklisted.BlacklistDatabase).C("ip").Find(bson.M{"index": data.Src}).All(&resList)
			if len(resList) > 0 {
				// set flag to true
				blacklistedSrc = true
				// build query
				uconnsQuery := getBlacklistsStatsQuery(data.Src, "src")
				// get stats
				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.UniqueConnTable).Pipe(uconnsQuery).One(&uconnStatsSrc)
			}

			// check if blacklisted destination
			var resList2 []ritaBLResult
			_ = ssn.DB(a.conf.S.Blacklisted.BlacklistDatabase).C("ip").Find(bson.M{"index": data.Dst}).All(&resList2)
			if len(resList2) > 0 {
				// set flag to true
				blacklistedDst = true
				// build query
				uconnsQuery := getBlacklistsStatsQuery(data.Dst, "dst")
				// get stats
				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.UniqueConnTable).Pipe(uconnsQuery).One(&uconnStatsDst)
			}

			// update src of connection in hosts table
			if isIPv4(data.Src) {
				var output update

				//if the connection has a blacklisted destination (the connection itself is a src though)
				if blacklistedDst {
					output = hasBlacklistedDstQuery(data, blacklistedSrc, uconnStatsSrc)
				} else { //otherwise, just add the result
					output = standardQuery(data.Src, data.IsLocalSrc, data.MaxDuration, data.TXTQueryCount, data.UntrustedAppConnCount, true, blacklistedSrc, uconnStatsSrc)
				}

				// set to writer channel
				a.analyzedCallback(output)
			}

			// update dst of connection in hosts table
			if isIPv4(data.Dst) {
				var output update

				//if the connection has a blacklisted source (the connection itself is a dst though)
				if blacklistedSrc {
					output = hasBlacklistedSrcQuery(data, blacklistedDst, uconnStatsDst)

				} else { //otherwise, just add the result
					output = standardQuery(data.Dst, data.IsLocalDst, data.MaxDuration, 0, 0, false, blacklistedDst, uconnStatsDst)
				}

				// set to writer channel
				a.analyzedCallback(output)
			}
		}
		a.analysisWg.Done()
	}()
}

//isIPv4 checks if an ip is ipv4
func isIPv4(address string) bool {
	return strings.Count(address, ":") < 2
}

//ipv4ToBinary generates binary representations of the IPv4 addresses
func ipv4ToBinary(ipv4 net.IP) int64 {
	return int64(binary.BigEndian.Uint32(ipv4[12:16]))
}

func (a *analyzer) isBlacklisted(host string) bool {
	ssn := a.db.Session.Copy()
	defer ssn.Close()

	// set blacklisted Flag
	blacklistFlag := false

	return blacklistFlag
}

//standardQuery ...
func standardQuery(ip string, local bool, maxdur float64, txtQCount int64, untrustedACC int64, src bool, blacklisted bool, uconnStats uconnRes) update {
	var output update

	// create query
	query := bson.M{
		"$setOnInsert": bson.M{
			"local":       local,
			"ipv4":        true,
			"ipv4_binary": ipv4ToBinary(net.ParseIP(ip)),
		},
		"$max": bson.M{"max_duration": maxdur},
	}

	if blacklisted {

		query["$set"] = bson.M{
			"blacklisted": true,
			"conn_count":  uconnStats.Connections,
			"uconn_count": uconnStats.UniqueConnections,
			"total_bytes": uconnStats.TotalBytes,
		}
	}

	if src {
		query["$inc"] = bson.M{
			"count_src":       1,
			"txt_query_count": txtQCount,
			"upps_count":      untrustedACC,
		}
	} else {
		query["$inc"] = bson.M{"count_dst": 1}
	}
	// create selector for output
	output.query = query
	output.selector = bson.M{"ip": ip}

	return output
}

//hasBlacklistedQuery ...
// If the internal system initiated the connection, then bl_out_count
// holds the number of unique blacklisted IPs the given host contacted.
func hasBlacklistedDstQuery(data *uconn.Pair, blacklisted bool, uconnStats uconnRes) update {

	var output update

	// create query
	query := bson.M{
		"$setOnInsert": bson.M{
			"local":       data.IsLocalSrc,
			"ipv4":        true,
			"ipv4_binary": ipv4ToBinary(net.ParseIP(data.Src)),
		},
		"$inc": bson.M{
			"count_src":       1,
			"bl_out_count":    data.ConnectionCount,
			"bl_total_bytes":  data.TotalBytes,
			"txt_query_count": data.TXTQueryCount,
			"upps_count":      data.UntrustedAppConnCount,
		},
		"$max": bson.M{"max_duration": data.MaxDuration},
	}

	if blacklisted {
		query["$set"] = bson.M{
			"blacklisted": true,
			"conn_count":  uconnStats.Connections,
			"uconn_count": uconnStats.UniqueConnections,
			"total_bytes": uconnStats.TotalBytes,
		}
	}

	// create selector for output
	output.query = query
	output.selector = bson.M{"ip": data.Src}

	return output
}

//hasBlacklistedSrcQuery ...
// If the blacklisted IP initiated the connection, then bl_in_count
// holds the number of unique IPs connected to the given
// host.
func hasBlacklistedSrcQuery(data *uconn.Pair, blacklisted bool, uconnStats uconnRes) update {
	var output update

	// create query
	query := bson.M{
		"$setOnInsert": bson.M{
			"local":       data.IsLocalDst,
			"ipv4":        true,
			"ipv4_binary": ipv4ToBinary(net.ParseIP(data.Dst)),
		},
		"$inc": bson.M{
			"count_dst":      1,
			"bl_in_count":    data.ConnectionCount,
			"bl_total_bytes": data.TotalBytes,
		},
		"$max": bson.M{"max_duration": data.MaxDuration},
	}

	if blacklisted {
		query["$set"] = bson.M{
			"blacklisted": true,
			"conn_count":  uconnStats.Connections,
			"uconn_count": uconnStats.UniqueConnections,
			"total_bytes": uconnStats.TotalBytes,
		}
	}

	// create selector for output
	output.query = query
	output.selector = bson.M{"ip": data.Dst}

	return output

}

//getBlacklistsStats will only run if an ip is determined to be a blacklisted ip
func getBlacklistsStatsQuery(host string, target string) []bson.M {
	//nolint: vet
	return []bson.M{
		bson.M{"$match": bson.M{target: host}},
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
