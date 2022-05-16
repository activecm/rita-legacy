package uconn

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
	//summarizer records summary data for individual hosts using unique connection data
	summarizer struct {
		chunk              int                // current chunk (0 if not on rolling summary)
		db                 *database.DB       // provides access to MongoDB
		conf               *config.Config     // contains details needed to access MongoDB
		log                *log.Logger        // main logger for RITA
		summarizedCallback func(update)       // called on each summarized result
		closedCallback     func()             // called when .close() is called and no more calls to summarizedCallback will be made
		summaryChannel     chan data.UniqueIP // holds unsummarized data
		summaryWg          sync.WaitGroup     // wait for summary to finish
	}
)

//newSummarizer creates a new summarizer for unique connection data
func newSummarizer(chunk int, db *database.DB, conf *config.Config, log *log.Logger, summarizedCallback func(update), closedCallback func()) *summarizer {
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

//collect sends a group of domains to be summarized
func (s *summarizer) collect(datum data.UniqueIP) {
	s.summaryChannel <- datum
}

//close waits for the collector to finish
func (s *summarizer) close() {
	close(s.summaryChannel)
	s.summaryWg.Wait()
	s.closedCallback()
}

//start kicks off a new summary thread
func (s *summarizer) start() {
	s.summaryWg.Add(1)
	go func() {

		ssn := s.db.Session.Copy()
		defer ssn.Close()

		for datum := range s.summaryChannel {
			uconnCollection := ssn.DB(s.db.GetSelectedDB()).C(s.conf.T.Structure.UniqueConnTable)
			hostCollection := ssn.DB(s.db.GetSelectedDB()).C(s.conf.T.Structure.HostTable)

			maxDurQuery, err := maxDurationQuery(datum, uconnCollection, s.chunk)
			if err != nil {
				s.log.WithFields(log.Fields{
					"Module": "uconns",
					"Data":   datum,
				}).Error(err)
				continue
			}

			totalHostQuery := maxDurQuery

			if len(totalHostQuery) > 0 {
				s.summarizedCallback(update{
					selector: datum.BSONKey(),
					query:    totalHostQuery,
				})
			}

			invalidCertUpdates, err := invalidCertUpdates(datum, uconnCollection, hostCollection, s.chunk)
			if err != nil {
				s.log.WithFields(log.Fields{
					"Module": "uconns",
					"Data":   datum,
				}).Error(err)
				continue
			}

			for _, update := range invalidCertUpdates {
				s.summarizedCallback(update)
			}
		}
		s.summaryWg.Done()
	}()
}

func maxDurationQuery(datum data.UniqueIP, uconnColl *mgo.Collection, chunk int) (bson.M, error) {
	var maxDurIP struct {
		Peer   data.UniqueIP `bson:"peer"`
		MaxDur float64       `bson:"maxdur"`
	}

	mdipQuery := maxDurationPipeline(datum, chunk)

	err := uconnColl.Pipe(mdipQuery).One(&maxDurIP)
	if err == mgo.ErrNotFound {
		return bson.M{}, nil
	}
	if err != nil {
		return bson.M{}, err
	}

	return bson.M{
		"$push": bson.M{
			"dat": bson.M{
				"$each": []bson.M{{
					"mdip":         maxDurIP.Peer,
					"max_duration": maxDurIP.MaxDur,
					"cid":          chunk,
				}},
			},
		},
	}, nil
}

func maxDurationPipeline(host data.UniqueIP, chunk int) []bson.M {
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
			// match uconn records which have been updated this chunk
			"dat.cid": chunk,
		}},
		// drop unecessary data
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
			"dat.cid":    1,
			"dat.maxdur": 1,
		}},
		// drop dat records that are not from this chunk
		{"$project": bson.M{
			"peer": 1,
			"dat": bson.M{"$filter": bson.M{
				"input": "$dat",
				"cond": bson.M{
					"$eq": []interface{}{"$$this.cid", chunk},
				},
			}},
		}},
		// for each peer, combine the records that match the current chunk
		{"$project": bson.M{
			"peer":   1,
			"maxdur": bson.M{"$max": "$dat.maxdur"},
		}},
		// find the peer with the maximum duration
		{"$sort": bson.M{
			"maxdur": -1,
		}},
		{"$limit": 1},
	}
}

func invalidCertUpdates(datum data.UniqueIP, uconnColl *mgo.Collection, hostColl *mgo.Collection, chunk int) ([]update, error) {

	var updates []update

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
			updates = append(updates, update{
				selector: hostEntryExistsSelector,
				query: bson.M{
					"$set": bson.M{
						"dat.$.cid": chunk,
					},
				},
			})
		} else {
			updates = append(updates, update{
				selector: datum.BSONKey(),
				query: bson.M{"$push": bson.M{
					"dat": bson.M{
						"icdst": icertPeer,
						"icert": 1,
						"cid":   chunk,
					},
				}},
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
