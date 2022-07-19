package host

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (
	//summarizer provides a summary of host records
	summarizer struct {
		chunk              int            //current chunk (0 if not on rolling summarize)
		conf               *config.Config // contains details needed to access MongoDB
		db                 *database.DB   // provides access to MongoDB
		log                *log.Logger    // logger for writing out errors and warnings
		summarizedCallback func(update)   // called on each summarized result
		closedCallback     func()         // called when .close() is called and no more calls to summarizedCallback will be made
		summarizeChannel   chan *Input    // holds data to be summarized
		summarizeWg        sync.WaitGroup // wait for summarize to finish
	}
)

//newSummarizer creates a new summarizer for host records
func newSummarizer(chunk int, conf *config.Config, db *database.DB, log *log.Logger, summarizedCallback func(update), closedCallback func()) *summarizer {
	return &summarizer{
		chunk:              chunk,
		conf:               conf,
		log:                log,
		db:                 db,
		summarizedCallback: summarizedCallback,
		closedCallback:     closedCallback,
		summarizeChannel:   make(chan *Input),
	}
}

//collect sends a chunk of data to be summarized
func (s *summarizer) collect(datum *Input) {
	s.summarizeChannel <- datum
}

//close waits for the collector to finish
func (s *summarizer) close() {
	close(s.summarizeChannel)
	s.summarizeWg.Wait()
	s.closedCallback()
}

//start kicks off a new summarize thread
func (s *summarizer) start() {
	s.summarizeWg.Add(1)
	go func() {
		ssn := s.db.Session.Copy()
		defer ssn.Close()

		for datum := range s.summarizeChannel {
			hostCollection := ssn.DB(s.db.GetSelectedDB()).C(s.conf.T.Structure.HostTable)
			expDNSSummaryUpdate := maxExplodedDNSQuery(datum, hostCollection, s.chunk)

			totalUpdate := expDNSSummaryUpdate

			s.summarizedCallback(update{
				selector: datum.Host.BSONKey(),
				query:    totalUpdate,
			})
		}
		s.summarizeWg.Done()
	}()
}

//maxExplodedDNSQuery records the exploded DNS super domain with the most queries
//from the given host. The whole observation period is aggregated, not just the current chunk.
func maxExplodedDNSQuery(datum *Input, hostColl *mgo.Collection, chunk int) bson.M {
	// If there aren't any explodedDNS results, max_dns will be set to
	// {"query": "", count: 0}.
	var maxDNSQueryCount explodedDNS
	hostColl.Pipe(maxDNSQueryCountPipeline(datum.Host)).One(&maxDNSQueryCount)

	return bson.M{
		"$push": bson.M{
			"dat": bson.M{
				"max_dns": maxDNSQueryCount,
				"cid":     chunk,
			},
		},
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
func maxDNSQueryCountPipeline(host data.UniqueIP) []bson.M {
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
