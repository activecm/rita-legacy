package host

import (
	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/data"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"strconv"
	"strings"
	"sync"
)

type (
	//analyzer : structure for host analysis
	analyzer struct {
		chunk            int            //current chunk (0 if not on rolling analysis)
		chunkStr         string         //current chunk (0 if not on rolling analysis)
		conf             *config.Config // contains details needed to access MongoDB
		db               *database.DB   // provides access to MongoDB
		log              *logrus.Logger // logger for writing out errors and warnings
		analyzedCallback func(update)   // called on each analyzed result
		closedCallback   func()         // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Input    // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for gathering data
func newAnalyzer(chunk int, conf *config.Config, db *database.DB, log *logrus.Logger, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		chunkStr:         strconv.Itoa(chunk),
		conf:             conf,
		log:              log,
		db:               db,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *Input),
	}
}

//collect sends a chunk of data to be analyzed
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
			// blacklisted flag
			blacklisted := false

			// check if blacklisted destination
			blCount, _ := ssn.DB(a.conf.S.Blacklisted.BlacklistDatabase).C("ip").Find(bson.M{"index": datum.Host.IP}).Count()
			if blCount > 0 {
				blacklisted = true
			}

			// update src of connection in hosts table
			if datum.IP4 {
				var output update

				newRecordFlag := a.shouldInsertNewHostRecord(ssn, datum.Host)

				var maxDNSQueryRes explodedDNS
				// If we have any dns queries for this host, push them to the database
				// and retrieve the max dns query count object.
				// If there aren't any explodedDNS results, max_dns will be set to
				// {"query": "", count: 0}.
				if len(datum.DNSQueryCount) > 0 {
					// update the host record with the new exploded dns results
					explodedDNSEntries := buildExplodedDNSArray(datum.DNSQueryCount)
					a.writeExplodedDNSEntries(ssn, datum.Host, explodedDNSEntries, newRecordFlag)

					// determine the  max dns query count query
					maxDNSQuery := maxDNSQueryCountQuery(datum.Host)
					err := ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Pipe(maxDNSQuery).AllowDiskUse().One(&maxDNSQueryRes)
					// log erros
					if err != nil {
						a.log.WithFields(log.Fields{
							"Module": "host",
							"Data":   maxDNSQuery,
						}).Error(err)
					}
				}

				output = standardQuery(a.chunk, a.chunkStr, datum.Host, datum.IsLocal, datum.IP4, datum.IP4Bin, datum.MaxDuration, maxDNSQueryRes, datum.UntrustedAppConnCount, datum.CountSrc, datum.CountDst, blacklisted, newRecordFlag)

				// set to writer channel
				a.analyzedCallback(output)

			}

		}
		a.analysisWg.Done()
	}()
}

//shouldInsertNewRecord returns true if a host entry with the current CID does not exist in the database
func (a *analyzer) shouldInsertNewHostRecord(ssn *mgo.Session, host data.UniqueIP) bool {
	var hostCIDs []struct {
		CID int `bson:"cid"`
	}

	_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(host.BSONKey()).All(&hostCIDs)

	if len(hostCIDs) <= 0 || hostCIDs[0].CID != a.chunk {
		// fmt.Println("host no results", len(hostCIDs), host.IP)
		return true
	}
	return false
}

//buildExplodedDNSArray generates exploded dns query results given how many times each full fqdn
//was queried. Returns the results as an array for MongoDB compatibility
func buildExplodedDNSArray(dnsQueryCounts map[string]int64) []explodedDNS {
	// make a new map to store the exploded dns query->count data
	var explodedDNSMap map[string]int64
	explodedDNSMap = make(map[string]int64)
	for domain := range dnsQueryCounts {
		// split name on periods
		split := strings.Split(domain, ".")

		// we will not count the very last item, because it will be either all or
		// a part of the tlds. This means that something like ".co.uk" will still
		// not be fully excluded, but it will greatly reduce the complexity for the
		// most common tlds
		max := len(split) - 1

		for i := 1; i <= max; i++ {
			// parse domain which will be the part we are on until the end of the string
			entry := strings.Join(split[max-i:], ".")
			explodedDNSMap[entry]++
		}
	}

	// put exploded dns map into mongo format so that we can push the entire
	// exploded dns map data into the database in one go
	var explodedDNSEntries []explodedDNS
	for domain, count := range explodedDNSMap {
		var explodedDNSEntry explodedDNS
		explodedDNSEntry.Query = domain
		explodedDNSEntry.Count = count
		explodedDNSEntries = append(explodedDNSEntries, explodedDNSEntry)
	}
	return explodedDNSEntries
}

//writeExplodedDNSEntries pushes the explodedDNS results for the current import session into a host entry int the database
func (a *analyzer) writeExplodedDNSEntries(ssn *mgo.Session, host data.UniqueIP, explodedDNSEntries []explodedDNS, newRecordFlag bool) {

	// push the host exploded dns results into this host's dat array
	var input update
	// create selectors for input
	// if this is a new host, only use the host as the selector
	if newRecordFlag {
		input.selector = host.BSONKey()
	} else {
		// if this is an existing host, use the host & cid as the selectors
		input.selector = host.BSONKey()
		input.selector["dat.cid"] = a.chunk
	}
	input.query = bson.M{
		"$push": bson.M{
			"dat": bson.M{
				"exploded_dns": explodedDNSEntries,
				"cid":          a.chunk,
			},
		},
	}

	// upsert these exploded dns entries
	info, err := ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Upsert(input.selector, input.query)

	// log errors
	if err != nil ||
		((info.Updated == 0) && (info.UpsertedId == nil)) {
		a.log.WithFields(log.Fields{
			"Module": "host",
			"Info":   info,
			"Data":   input,
		}).Error(err)
	}
}

// db.getCollection('host').aggregate([
//     {"$match": {
//         "ip": "HOST IP",
//         "network_uuid": UUID(),
//     }},
//     {"$unwind": "$dat"},
//     {"$unwind": "$dat.exploded_dns"},
//
//     {"$project": {
//         "exploded_dns": "$dat.exploded_dns"
//     }},
//     {"$group": {
//         "_id": "$exploded_dns.query",
// 				 "query": {"$first": "$exploded_dns.query"}
//         "count": {"$sum": "$exploded_dns.count"}
//     }},
//     {"$project": {
//      	"_id": 0,
// 	      "query": 1,
// 	      "count": 1,
//     }},
//     {"$sort": {"count": -1}},
//     {"$limit": 1}
// ])
func maxDNSQueryCountQuery(host data.UniqueIP) []bson.M {
	query := []bson.M{
		{"$match": bson.M{
			"ip":           host.IP,
			"network_uuid": host.NetworkUUID,
		}},
		{"$unwind": "$dat"},
		{"$unwind": "$dat.exploded_dns"},
		{"$project": bson.M{
			"exploded_dns": "$dat.exploded_dns",
		}},
		{"$group": bson.M{
			"_id":   "$exploded_dns.query",
			"query": bson.M{"$first": "$exploded_dns.query"},
			"count": bson.M{"$sum": "$exploded_dns.count"},
		}},
		{"$project": bson.M{
			"_id":   0,
			"query": 1,
			"count": 1,
		}},
		{"$sort": bson.M{"count": -1}},
		{"$limit": 1},
	}
	return query
}

//standardQuery ...
func standardQuery(chunk int, chunkStr string, ip data.UniqueIP, local bool, ip4 bool, ip4bin int64, maxdur float64, maxDNSQueryCount explodedDNS, untrustedACC int64, countSrc int, countDst int, blacklisted bool, newFlag bool) update {
	var output update

	// create query
	query := bson.M{
		"$set": bson.M{
			"blacklisted":  blacklisted,
			"cid":          chunk,
			"local":        local,
			"ipv4":         ip4,
			"ipv4_binary":  ip4bin,
			"network_name": ip.NetworkName,
		},
	}
	if newFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"$each": []bson.M{
					{
						"count_src":  countSrc,
						"count_dst":  countDst,
						"upps_count": untrustedACC,
						"cid":        chunk,
					},
					{
						"max_dns": maxDNSQueryCount,
						"cid":     chunk,
					},
				},
			}}

		// create selector for output ,
		output.query = query
		output.selector = ip.BSONKey()

	} else {

		query["$inc"] = bson.M{
			"dat.$.count_src":  countSrc,
			"dat.$.count_dst":  countDst,
			"dat.$.upps_count": untrustedACC,
		}

		query["$push"] = bson.M{
			"dat": bson.M{
				"max_dns": maxDNSQueryCount,
				"cid":     chunk,
			},
		}

		// create selector for output
		output.query = query
		output.selector = ip.BSONKey()
		output.selector["dat.cid"] = chunk
	}

	return output
}
