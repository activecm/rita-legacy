package useragent

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
	//summarizer records summary data of rare signatures for individual hosts
	summarizer struct {
		chunk              int                        // current chunk (0 if not on rolling summary)
		db                 *database.DB               // provides access to MongoDB
		conf               *config.Config             // contains details needed to access MongoDB
		log                *log.Logger                // main logger for RITA
		summarizedCallback func(database.BulkChanges) // called on each summarized result
		closedCallback     func()                     // called when .close() is called and no more calls to summarizedCallback will be made
		summaryChannel     chan *Input                // holds unsummarized data
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
		summaryChannel:     make(chan *Input),
	}
}

// collect gathers a useragent to summarize
func (s *summarizer) collect(datum *Input) {
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

			// if there are too many IPs associated with this signature in the parsed in files, ignore the
			// rare signature host collection update
			if len(datum.OrigIps) >= rareSignatureOrigIPsCutoff {
				continue
			}

			// get the origIPs from the database for the given user agent
			// since the useragent collection update may not have taken place yet, we need to union in
			// the new origIPs from the recently parsed in files
			dbRareSigOrigIPs, err := getOrigIPsForAgentFromDB(useragentCollection, datum.Name)
			if err != nil {
				s.log.WithFields(log.Fields{
					"Module": "useragent",
					"Data":   datum,
				}).Error(err)
				continue
			}

			// if there are too many IPs associated with this signature in the database, ignore the
			// rare signature host collection update
			if len(dbRareSigOrigIPs) >= rareSignatureOrigIPsCutoff {
				continue
			}

			// merge the two lists of origIPs to get rid of duplicates
			origIPsUnioned := unionUniqueIPSlices(datum.OrigIps.Items(), dbRareSigOrigIPs)

			// if we've busted over the limit after unioning the new and old originating IPs together
			// don't update the host records
			if len(origIPsUnioned) >= rareSignatureOrigIPsCutoff {
				continue
			}

			rareSignatureUpdates, err := rareSignatureUpdates(datum.OrigIps.Items(), datum.Name, hostCollection, s.chunk)
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

// unionUniqueIPSlices merges two UniqueIP slices into one while removing duplicates.
func unionUniqueIPSlices(slice1 []data.UniqueIP, slice2 []data.UniqueIP) []data.UniqueIP {
	ipsUnionMap := make(map[string]data.UniqueIP)
	for i := range slice1 {
		ipsUnionMap[slice1[i].MapKey()] = slice1[i]
	}
	for i := range slice2 {
		ipsUnionMap[slice2[i].MapKey()] = slice2[i]
	}

	ipsUnionSlice := make([]data.UniqueIP, 0, len(ipsUnionMap))

	for _, ip := range ipsUnionMap {
		ipsUnionSlice = append(ipsUnionSlice, ip)
	}
	return ipsUnionSlice
}

/*
db.getCollection('useragent').aggregate([
    {"$match": {"user_agent": "Mozilla/5.0 (Windows NT; Windows NT 10.0; en-US) WindowsPowerShell/5.1.16299.248"}},
    {"$project": {"ips": "$dat.orig_ips"}},
    {"$unwind": "$ips"},
    {"$unwind": "$ips"},
    {"$group": {
            "_id": {
                    "ip" : "$ips.ip",
                    "network_uuid": "$ips.network_uuid",
            },
            "network_name": {"$last": "$ips.network_name"},
    }},
    {"$project": {
		"_id": 0,
        "ip":           "$_id.ip",
        "network_uuid": "$_id.network_uuid",
        "network_name": "$network_name",
    }},
])
*/
//getOrigIPsForAgentFromDB returns the originating IPs associated witha given useragent from the database
func getOrigIPsForAgentFromDB(useragentCollection *mgo.Collection, name string) ([]data.UniqueIP, error) {
	query := []bson.M{
		{"$match": bson.M{"user_agent": name}},
		{"$project": bson.M{"ips": "$dat.orig_ips"}},
		{"$unwind": "$ips"},
		{"$unwind": "$ips"}, // not an error, needs to be done twice
		{"$group": bson.M{
			"_id": bson.M{
				"ip":           "$ips.ip",
				"network_uuid": "$ips.network_uuid",
			},
			"network_name": bson.M{"$last": "$ips.network_name"},
		}},
		{"$project": bson.M{
			"_id":          0,
			"ip":           "$_id.ip",
			"network_uuid": "$_id.network_uuid",
			"network_name": "$network_name",
		}},
	}

	var dbRareSigOrigIPs []data.UniqueIP

	err := useragentCollection.Pipe(query).AllowDiskUse().All(&dbRareSigOrigIPs)

	return dbRareSigOrigIPs, err
}

// rareSignatureUpdates formats MongoDB update for each internal host which either inserts a new rare signature host
// record into that host's dat array in the host collection or updates an existing
// record in the host's dat array for the rare signature with the current chunk id.
func rareSignatureUpdates(rareSigIPs []data.UniqueIP, signature string, hostCollection *mgo.Collection, chunk int) ([]database.BulkChange, error) {
	var updates []database.BulkChange

	for _, rareSigIP := range rareSigIPs {
		hostEntryExistsSelector := rareSigIP.BSONKey()
		hostEntryExistsSelector["dat.rsig"] = signature

		nExistingEntries, err := hostCollection.Find(hostEntryExistsSelector).Count()
		if err != nil {
			return updates, err
		}

		if nExistingEntries > 0 {
			// no need to update all of the fields for an existing
			// record; we just need to update the chunk ID
			updates = append(updates, database.BulkChange{
				Selector: hostEntryExistsSelector,
				Update: bson.M{
					"$set": bson.M{
						"dat.$.cid": chunk,
					},
				},
				Upsert: true,
			})
		} else {
			// add a new rare signature entry
			updates = append(updates, database.BulkChange{
				Selector: rareSigIP.BSONKey(),
				Update: bson.M{
					"$push": bson.M{
						"dat": bson.M{
							"$each": []bson.M{{
								"rsig":  signature,
								"rsigc": 1,
								"cid":   chunk,
							}},
						},
					},
				},
				Upsert: true,
			})
		}
	}

	return updates, nil
}
