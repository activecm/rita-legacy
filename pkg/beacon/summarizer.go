package beacon

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
	//summarizer records summary data for individual hosts using beacon data
	summarizer struct {
		chunk              int                  // current chunk (0 if not on rolling summary)
		db                 *database.DB         // provides access to MongoDB
		conf               *config.Config       // contains details needed to access MongoDB
		log                *log.Logger          // main logger for RITA
		summarizedCallback func(mgoBulkActions) // called on each summarized result
		closedCallback     func()               // called when .close() is called and no more calls to summarizedCallback will be made
		summaryChannel     chan data.UniqueIP   // holds unsummarized data
		summaryWg          sync.WaitGroup       // wait for summary to finish
	}
)

//newSummarizer creates a new summarizer for beacon data
func newSummarizer(chunk int, db *database.DB, conf *config.Config, log *log.Logger, summarizedCallback func(mgoBulkActions), closedCallback func()) *summarizer {
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

//collect collects an internal host to create summary data for
func (s *summarizer) collect(datum data.UniqueIP) {
	s.summaryChannel <- datum
}

//close waits for the summarizer to finish
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
			beaconCollection := ssn.DB(s.db.GetSelectedDB()).C(s.conf.T.Beacon.BeaconTable)

			maxBeaconQuery, err := maxBeaconQuery(datum, beaconCollection, s.chunk)
			if err != nil {
				if err != mgo.ErrNotFound {
					s.log.WithFields(log.Fields{
						"Module": "beacon",
						"Data":   datum,
					}).Error(err)
				}
				continue
			}

			// copy variables to be used by bulk callback to prevent capturing by reference
			hostSelector := datum.BSONKey()
			totalHostQuery := maxBeaconQuery
			if len(totalHostQuery) > 0 {
				s.summarizedCallback(mgoBulkActions{
					s.conf.T.Structure.HostTable: func(b *mgo.Bulk) int {
						b.Upsert(hostSelector, totalHostQuery)
						return 1
					},
				})
			}
		}
		s.summaryWg.Done()
	}()
}

//maxBeaconQuery finds the highest scoring beacon from this import session for a particular host
func maxBeaconQuery(datum data.UniqueIP, beaconColl *mgo.Collection, chunk int) (bson.M, error) {

	var maxBeaconIP struct {
		Dst   data.UniqueIP `bson:"dst"`
		Score float64       `bson:"score"`
	}

	mbdstQuery := maxBeaconPipeline(datum, chunk)
	err := beaconColl.Pipe(mbdstQuery).One(&maxBeaconIP)
	if err != nil {
		return bson.M{}, err
	}

	return bson.M{
		"$push": bson.M{
			"dat": bson.M{
				"$each": []bson.M{{
					"mbdst":            maxBeaconIP.Dst,
					"max_beacon_score": maxBeaconIP.Score,
					"cid":              chunk,
				}},
			},
		},
	}, nil
}

func maxBeaconPipeline(host data.UniqueIP, chunk int) []bson.M {
	return []bson.M{
		{"$match": bson.M{
			"src":              host.IP,
			"src_network_uuid": host.NetworkUUID,
			"cid":              chunk,
		}},
		// drop unnecessary data
		{"$project": bson.M{
			"dst": bson.M{
				"ip":           "$dst",
				"network_uuid": "$dst_network_uuid",
				"network_name": "$dst_network_name",
			},
			"score": 1,
		}},
		// find the peer with the maximum score
		{"$sort": bson.M{
			"score": -1,
		}},
		{"$limit": 1},
	}
}
