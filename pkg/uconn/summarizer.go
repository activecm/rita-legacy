package uconn

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
	//summarizer records summary data for individual hosts using unique connection data
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

// collect collects an internal host to be summarized
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
			uconnCollection := ssn.DB(s.db.GetSelectedDB()).C(s.conf.T.Structure.UniqueConnTable)
			hostCollection := ssn.DB(s.db.GetSelectedDB()).C(s.conf.T.Structure.HostTable)

			hostUpdates := []database.BulkChange{}

			maxTotalDurUpdate, err := maxTotalDurationUpdate(datum, uconnCollection, hostCollection, s.chunk)
			if err != nil {
				if err != mgo.ErrNotFound {
					s.log.WithFields(log.Fields{
						"Module": "uconns",
						"Data":   datum,
					}).Error(err)
				}
				continue
			}
			if maxTotalDurUpdate.Selector != nil {
				hostUpdates = append(hostUpdates, maxTotalDurUpdate)
			}

			invalidCertUpdates, err := invalidCertUpdates(datum, uconnCollection, hostCollection, s.chunk)
			if err != nil {
				s.log.WithFields(log.Fields{
					"Module": "uconns",
					"Data":   datum,
				}).Error(err)
				continue
			}
			hostUpdates = append(hostUpdates, invalidCertUpdates...)

			if len(hostUpdates) > 0 {
				s.summarizedCallback(database.BulkChanges{
					s.conf.T.Structure.HostTable: hostUpdates,
				})
			}
		}
		s.summaryWg.Done()
	}()
}

func maxTotalDurationUpdate(datum data.UniqueIP, uconnColl, hostColl *mgo.Collection, chunk int) (database.BulkChange, error) {
	var maxDurIP struct {
		Peer        data.UniqueIP `bson:"peer"`
		MaxTotalDur float64       `bson:"tdur"`
	}

	mdipQuery := maxTotalDurationPipeline(datum)

	err := uconnColl.Pipe(mdipQuery).One(&maxDurIP)
	if err != nil {
		return database.BulkChange{}, err
	}

	hostSelector := datum.BSONKey()
	hostWithDatEntrySelector := database.MergeBSONMaps(
		hostSelector,
		bson.M{"dat": bson.M{"$elemMatch": bson.M{"mdip": bson.M{"$exists": true}}}},
	)

	nExistingEntries, err := hostColl.Find(hostWithDatEntrySelector).Count()
	if err != nil {
		return database.BulkChange{}, err
	}

	if nExistingEntries > 0 {
		updateQuery := bson.M{
			"$set": bson.M{
				"dat.$.mdip":         maxDurIP.Peer,
				"dat.$.max_duration": maxDurIP.MaxTotalDur,
				"dat.$.cid":          chunk,
			},
		}
		return database.BulkChange{Selector: hostWithDatEntrySelector, Update: updateQuery, Upsert: true}, nil
	}

	insertQuery := bson.M{
		"$push": bson.M{
			"dat": bson.M{
				// NOTE: While "max_total_duration" would be a better name for this database field,
				// "max_duration" is used to preserve database schema compatibility.
				// This analysis previously tracked the longest individual connection for each internal host
				// and stored the result in the `host` collection with the key `dat.max_duration`.
				"$each": []bson.M{{
					"mdip":         maxDurIP.Peer,
					"max_duration": maxDurIP.MaxTotalDur,
					"cid":          chunk,
				}},
			},
		},
	}
	return database.BulkChange{Selector: hostSelector, Update: insertQuery, Upsert: true}, nil
}

func maxTotalDurationPipeline(host data.UniqueIP) []bson.M {
	return []bson.M{
		{"$match": bson.M{
			// match the host IP/ network
			"$expr": bson.M{
				"$or": []bson.M{
					{"$and": []bson.M{
						{"$eq": []interface{}{"$src", host.IP}},
						{"$eq": []interface{}{"$src_network_uuid", host.NetworkUUID}},
					}},
					{"$and": []bson.M{
						{"$eq": []interface{}{"$dst", host.IP}},
						{"$eq": []interface{}{"$dst_network_uuid", host.NetworkUUID}},
					}},
				},
			},
		}},
		// drop unnecessary data
		{"$project": bson.M{
			"peer": bson.M{
				"ip": bson.M{
					"$cond": bson.M{
						"if":   bson.M{"$eq": []interface{}{"dst", host.IP}},
						"then": "$src",
						"else": "$dst",
					},
				},
				"network_uuid": bson.M{
					"$cond": bson.M{
						"if":   bson.M{"$eq": []interface{}{"dst_network_uuid", host.NetworkUUID}},
						"then": "$src_network_uuid",
						"else": "$dst_network_uuid",
					},
				},
				"network_name": bson.M{
					"$cond": bson.M{
						"if":   bson.M{"$eq": []interface{}{"dst_network_name", host.NetworkName}},
						"then": "$src_network_name",
						"else": "$dst_network_name",
					},
				},
			},
			"dat.tdur": 1,
		}},
		// for each peer, combine the records
		{"$project": bson.M{
			"peer": 1,
			"tdur": bson.M{"$sum": "$dat.tdur"},
		}},
		// find the peer with the maximum duration
		{"$sort": bson.M{
			"tdur": -1,
		}},
		{"$limit": 1},
	}
}

func invalidCertUpdates(datum data.UniqueIP, uconnColl *mgo.Collection, hostColl *mgo.Collection, chunk int) ([]database.BulkChange, error) {

	var updates []database.BulkChange

	icertQuery := invalidCertPipeline(datum, chunk)
	var icertPeer data.UniqueIP
	icertPeerIter := uconnColl.Pipe(icertQuery).Iter()
	for icertPeerIter.Next(&icertPeer) {
		hostEntryExistsSelector := datum.BSONKey()
		hostEntryExistsSelector["dat"] = bson.M{"$elemMatch": icertPeer.PrefixedBSONKey("icdst")}
		nExistingEntries, err := hostColl.Find(hostEntryExistsSelector).Count()
		if err != nil {
			return updates, err
		}

		if nExistingEntries > 0 {
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
			updates = append(updates, database.BulkChange{
				Selector: datum.BSONKey(),
				Update: bson.M{"$push": bson.M{
					"dat": bson.M{
						"$each": []bson.M{{
							"icdst": icertPeer,
							"icert": 1,
							"cid":   chunk,
						}},
					},
				}},
				Upsert: true,
			})
		}
	}
	if err := icertPeerIter.Close(); err != nil {
		return updates, err
	}

	return updates, nil
}

func invalidCertPipeline(host data.UniqueIP, chunk int) []bson.M {
	return []bson.M{
		{"$match": bson.M{
			// match the host IP/ network
			"src":              host.IP,
			"src_network_uuid": host.NetworkUUID,
			"dat": bson.M{
				"$elemMatch": bson.M{
					"cid":    chunk,
					"icerts": true,
				},
			},
		}},
		// drop unnecessary data
		{"$project": bson.M{
			"ip":           "$dst",
			"network_uuid": "$dst_network_uuid",
			"network_name": "$dst_network_name",
		}},
	}
}
