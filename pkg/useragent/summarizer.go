package useragent

import (
	"sync"

	"github.com/activecm/rita-legacy/config"
	"github.com/activecm/rita-legacy/database"
	"github.com/activecm/rita-legacy/pkg/data"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (
	//summarizer records summary data of rare signatures for individual hosts
	summarizer struct {
		chunk              int                        // current chunk (0 if not on rolling summary)
		db                 *database.DB               // provides access to MongoDB
		conf               *config.Config             // contains details needed to access MongoDB
		log                *log.Logger                // main logger for RITA
		summarizedCallback func(database.BulkChanges) // called on each summarized result
		closedCallback     func()                     // called when .close() is called and no more calls to summarizedCallback will be made
		summaryChannel     chan data.UniqueIP         // holds unsummarized data
		summaryWg          sync.WaitGroup             // wait for summary to finish
	}
)

// newSummarizer creates a new summarizer for unique connection data
func newSummarizer(chunk int, db *database.DB, conf *config.Config, log *log.Logger, summarizedCallback func(database.BulkChanges), closedCallback func()) *summarizer {
	return &summarizer{
		chunk:              chunk,
		db:                 db,
		conf:               conf,
		log:                log,
		summarizedCallback: summarizedCallback,
		closedCallback:     closedCallback,
		summaryChannel:     make(chan data.UniqueIP),
	}
}

// collect gathers a useragent to summarize
func (s *summarizer) collect(datum data.UniqueIP) {
	s.summaryChannel <- datum
}

// close waits for the summarizer to finish
func (s *summarizer) close() {
	close(s.summaryChannel)
	s.summaryWg.Wait()
	s.closedCallback()
}

// start kicks off a new summary thread
func (s *summarizer) start() {
	s.summaryWg.Add(1)
	go func() {

		ssn := s.db.Session.Copy()
		defer ssn.Close()

		for datum := range s.summaryChannel {
			useragentCollection := ssn.DB(s.db.GetSelectedDB()).C(s.conf.T.UserAgent.UserAgentTable)
			hostCollection := ssn.DB(s.db.GetSelectedDB()).C(s.conf.T.Structure.HostTable)

			rareSignatures, err := getRareSignaturesForIP(useragentCollection, datum, s.chunk)
			if err != nil {
				s.log.WithFields(log.Fields{
					"Module": "useragent",
					"Data":   datum,
				}).Error(err)
				continue
			}
			if len(rareSignatures) == 0 {
				continue // nothing to update
			}

			rareSignatureUpdates, err := rareSignatureUpdates(datum, rareSignatures, hostCollection, s.chunk)
			if err != nil {
				s.log.WithFields(log.Fields{
					"Module": "useragent",
					"Data":   datum,
				}).Error(err)
				continue
			}

			if len(rareSignatureUpdates) > 0 {
				s.summarizedCallback(database.BulkChanges{s.conf.T.Structure.HostTable: rareSignatureUpdates})
			}
		}
		s.summaryWg.Done()
	}()
}

/*
db.getCollection('useragent').aggregate([

	{"$match": {
	    "dat": {
	        "$elemMatch": {
	            "orig_ips.ip": "10.55.200.11",
	            "cid": 0,
	        }
	    }
	}},
	{"$project": {
	    "user_agent": 1,
	    "ips": "$dat.orig_ips",
	}},
	{"$unwind": "$ips"},
	{"$unwind": "$ips"},
	{"$group": {
	    "_id": {
	        "user_agent": "$user_agent",
	        "ip":           "$ips.ip",
	        "network_uuid": "$ips.network_uuid",
	    },
	    "network_name": {"$last": "$ips.network_name"},
	}},
	{"$group": {
	    "_id": {
	        "user_agent": "$_id.user_agent",
	    },
	    "ips": {"$push": {
	        "ip": "$_id.ip",
	        "network_uuid": "$_id.network_uuid",
	        "network_name": "$network_name",
	    }},
	    "ips_count": {"$sum": 1},
	}},
	{"$match": {
	    "ips_count": {"$lt": 5},
	}},
	{"$project": {
	    "_id": 0,
	    "user_agent": "$_id.user_agent"
	}}

])
*/
func getRareSignaturesForIP(useragentCollection *mgo.Collection, host data.UniqueIP, chunk int) ([]string, error) {
	query := []bson.M{
		{"$match": bson.M{
			"dat": bson.M{
				"$elemMatch": database.MergeBSONMaps(
					host.PrefixedBSONKey("orig_ips"),
					bson.M{"cid": chunk},
				),
			},
		}},
		{"$project": bson.M{
			"user_agent": 1,
			"ips":        "$dat.orig_ips",
		}},
		{"$unwind": "$ips"},
		{"$unwind": "$ips"},
		{"$group": bson.M{
			"_id": bson.M{
				"user_agent":   "$user_agent",
				"ip":           "$ips.ip",
				"network_uuid": "$ips.network_uuid",
			},
			"network_name": bson.M{"$last": "$ips.network_name"},
		}},
		{"$group": bson.M{
			"_id": bson.M{
				"user_agent": "$_id.user_agent",
			},
			"ips": bson.M{"$push": bson.M{
				"ip":           "$_id.ip",
				"network_uuid": "$_id.network_uuid",
				"network_name": "$network_name",
			}},
			"ips_count": bson.M{"$sum": 1},
		}},
		{"$match": bson.M{
			"ips_count": bson.M{"$lt": rareSignatureOrigIPsCutoff},
		}},
		{"$project": bson.M{
			"_id":        0,
			"user_agent": "$_id.user_agent",
		}},
	}

	var aggResults []string
	var aggResult Result
	aggIter := useragentCollection.Pipe(query).Iter()
	for aggIter.Next(&aggResult) {
		aggResults = append(aggResults, aggResult.UserAgent)

	}
	if aggIter.Err() != nil && aggIter.Err() != mgo.ErrNotFound {
		return []string{}, aggIter.Err()
	}
	return aggResults, nil
}

// rareSignatureUpdates formats a MongoDB update for an internal host which either inserts a
// new rare signature host records into that host's dat array in the host collection or updates the
// existing records in the host's dat array for each rare signature with the current chunk id.
func rareSignatureUpdates(rareSigIP data.UniqueIP, newSignatures []string, hostCollection *mgo.Collection, chunk int) ([]database.BulkChange, error) {
	var updates []database.BulkChange

	existingRareSignaturesQuery := []bson.M{
		{"$match": rareSigIP.BSONKey()},
		{"$unwind": "$dat"},
		{"$match": bson.M{"dat.rsig": bson.M{"$exists": true}}},
		{"$project": bson.M{"user_agent": "$dat.rsig"}},
	}

	var existingSigs []Result
	err := hostCollection.Pipe(existingRareSignaturesQuery).AllowDiskUse().All(&existingSigs)
	if err != nil {
		return updates, err
	}

	// place existing signatures in a map to make the cross lookup fast
	existingSigsMap := make(map[string]struct{})
	for _, sig := range existingSigs {
		existingSigsMap[sig.UserAgent] = struct{}{}
	}

	// generate an update for each existing rare signature.

	// Unfortunately, there isn't a way to batch these into
	// a single update with the current MongoDB driver.
	// Normally, we could use array filters, but the bulk api doesn't
	// support updates with array filters.
	for _, sig := range newSignatures {
		if _, ok := existingSigsMap[sig]; ok {
			updates = append(updates, database.BulkChange{
				Selector: database.MergeBSONMaps(
					rareSigIP.BSONKey(),
					bson.M{"dat.rsig": sig},
				),
				Update: bson.M{
					"$set": bson.M{
						"dat.$.cid": chunk,
					},
				},
				Upsert: true,
			})
		}
	}

	// generate a single update for all of the new signatures
	var newSigDatSubdocs []bson.M
	for _, sig := range newSignatures {
		if _, ok := existingSigsMap[sig]; !ok {
			newSigDatSubdocs = append(newSigDatSubdocs, bson.M{
				"rsig":  sig,
				"rsigc": 1,
				"cid":   chunk,
			})
		}
	}
	// format update to push all of the new signature subdocuments
	if len(newSigDatSubdocs) > 0 {
		updates = append(updates, database.BulkChange{
			Selector: rareSigIP.BSONKey(),
			Update: bson.M{
				"$push": bson.M{
					"dat": bson.M{
						"$each": newSigDatSubdocs,
					},
				},
			},
			Upsert: true,
		})
	}

	return updates, nil
}
