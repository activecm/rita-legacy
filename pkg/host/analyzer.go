package host

import (
	"encoding/binary"
	"net"
	"strconv"
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
		chunk            int              //current chunk (0 if not on rolling analysis)
		chunkStr         string           //current chunk (0 if not on rolling analysis)
		db               *database.DB     // provides access to MongoDB
		conf             *config.Config   // contains details needed to access MongoDB
		analyzedCallback func(update)     // called on each analyzed result
		closedCallback   func()           // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *uconn.Pair // holds unanalyzed data
		analysisWg       sync.WaitGroup   // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for gathering data
func newAnalyzer(chunk int, db *database.DB, conf *config.Config, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		chunkStr:         strconv.Itoa(chunk),
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

			// check if blacklisted destination
			var resList2 []ritaBLResult
			_ = ssn.DB(a.conf.S.Blacklisted.BlacklistDatabase).C("ip").Find(bson.M{"index": data.Dst}).All(&resList2)
			if len(resList2) > 0 {
				blacklistedDst = true
			}

			// check if blacklisted source
			var resList []ritaBLResult
			_ = ssn.DB(a.conf.S.Blacklisted.BlacklistDatabase).C("ip").Find(bson.M{"index": data.Src}).All(&resList)
			if len(resList) > 0 {
				blacklistedSrc = true
			}

			// update src of connection in hosts table
			if isIPv4(data.Src) {
				var output update
				newRecordFlag := false
				var res2 []interface{}

				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(bson.M{"ip": data.Src}).All(&res2)

				if !(len(res2) > 0) {
					newRecordFlag = true
				}

				//if the connection has a blacklisted destination (the connection itself is a src though)
				if blacklistedDst {
					output = hasBlacklistedDstQuery(a.chunk, a.chunkStr, data, blacklistedSrc, newRecordFlag)
				} else { //otherwise, just add the result
					output = standardQuery(a.chunk, a.chunkStr, data.Src, data.IsLocalSrc, data.MaxDuration, data.TXTQueryCount, data.UntrustedAppConnCount, true, blacklistedSrc, newRecordFlag)
				}

				// set to writer channel
				a.analyzedCallback(output)
			}

			// update dst of connection in hosts table
			if isIPv4(data.Dst) {
				var output update

				newRecordFlag := false

				var res2 []interface{}

				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(bson.M{"ip": data.Dst}).All(&res2)

				if !(len(res2) > 0) {
					newRecordFlag = true
				}

				//if the connection has a blacklisted source (the connection itself is a dst though)
				if blacklistedSrc {
					output = hasBlacklistedSrcQuery(a.chunk, a.chunkStr, data, blacklistedDst, newRecordFlag)

				} else { //otherwise, just add the result
					output = standardQuery(a.chunk, a.chunkStr, data.Dst, data.IsLocalDst, data.MaxDuration, 0, 0, false, blacklistedDst, newRecordFlag)
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

//standardQuery ...
func standardQuery(chunk int, chunkStr string, ip string, local bool, maxdur float64, txtQCount int64, untrustedACC int64, src bool, blacklisted bool, newFlag bool) update {
	var output update

	// create query
	query := bson.M{
		"$setOnInsert": bson.M{
			"local":       local,
			"ipv4_binary": ipv4ToBinary(net.ParseIP(ip)),
		},

		"$set": bson.M{"blacklisted": blacklisted, "cid": chunk},
	}

	if newFlag {
		if src {
			query["$push"] = bson.M{
				"dat": bson.M{
					"count_src":       1,
					"txt_query_count": txtQCount,
					"upps_count":      untrustedACC,
					"max_duration":    maxdur,
					"cid":             chunk,
				}}
		} else {
			query["$push"] = bson.M{
				"dat": bson.M{
					"count_dst": 1,
					"cid":       chunk,
				}}
		}

	} else {
		if src {
			query["$inc"] = bson.M{
				"dat." + chunkStr + ".count_src":       1,
				"dat." + chunkStr + ".txt_query_count": txtQCount,
				"dat." + chunkStr + ".upps_count":      untrustedACC,
			}
		} else {
			query["$inc"] = bson.M{"dat." + chunkStr + ".count_dst": 1}
		}
		query["$max"] = bson.M{"dat." + chunkStr + ".max_duration": maxdur}
	}

	// create selector for output
	output.query = query
	output.selector = bson.M{"ip": ip}

	return output
}

//hasBlacklistedQuery ...
// If the internal system initiated the connection, then bl_out_count
// holds the number of unique blacklisted IPs the given host contacted.
func hasBlacklistedDstQuery(chunk int, chunkStr string, data *uconn.Pair, blacklisted bool, newFlag bool) update {

	var output update

	// create query
	query := bson.M{
		"$setOnInsert": bson.M{
			"local":       data.IsLocalSrc,
			"ipv4_binary": ipv4ToBinary(net.ParseIP(data.Src)),
		},
		"$set": bson.M{"blacklisted": blacklisted, "cid": chunk},
	}

	if newFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"count_src":       1,
				"bl_out_count":    data.ConnectionCount,
				"bl_total_bytes":  data.TotalBytes,
				"txt_query_count": data.TXTQueryCount,
				"upps_count":      data.UntrustedAppConnCount,
				"cid":             chunk,
				"max_duration":    data.MaxDuration,
			}}

	} else {

		query["$inc"] = bson.M{
			"dat." + chunkStr + ".count_src":       1,
			"dat." + chunkStr + ".bl_out_count":    data.ConnectionCount,
			"dat." + chunkStr + ".bl_total_bytes":  data.TotalBytes,
			"dat." + chunkStr + ".txt_query_count": data.TXTQueryCount,
			"dat." + chunkStr + ".upps_count":      data.UntrustedAppConnCount,
		}
		query["$max"] = bson.M{"dat." + chunkStr + ".max_duration": data.MaxDuration}
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
func hasBlacklistedSrcQuery(chunk int, chunkStr string, data *uconn.Pair, blacklisted bool, newFlag bool) update {
	var output update

	// create query
	query := bson.M{
		"$setOnInsert": bson.M{
			"local":       data.IsLocalDst,
			"ipv4_binary": ipv4ToBinary(net.ParseIP(data.Dst)),
		},
		"$set": bson.M{"blacklisted": blacklisted, "cid": chunk},
	}

	if newFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"count_dst":      1,
				"bl_in_count":    data.ConnectionCount,
				"bl_total_bytes": data.TotalBytes,
				"max_duration":   data.MaxDuration,
				"cid":            chunk,
			}}

	} else {

		query["$inc"] = bson.M{
			"dat." + chunkStr + ".count_dst":      1,
			"dat." + chunkStr + ".bl_in_count":    data.ConnectionCount,
			"dat." + chunkStr + ".bl_total_bytes": data.TotalBytes,
		}
		query["$max"] = bson.M{"dat." + chunkStr + ".max_duration": data.MaxDuration}
	}

	// create selector for output
	output.query = query
	output.selector = bson.M{"ip": data.Dst}

	return output

}
