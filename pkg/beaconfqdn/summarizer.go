package beaconfqdn

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
	//summarizer records summary data for individual hosts using fqdn beacon data
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

//newSummarizer creates a new summarizer for fqdn beacon data
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
			beaconFQDNCollection := ssn.DB(s.db.GetSelectedDB()).C(s.conf.T.BeaconFQDN.BeaconFQDNTable)

			maxBeaconQuery, err := maxFQDNBeaconQuery(datum, beaconFQDNCollection, s.chunk)
			if err != nil {
				if err != mgo.ErrNotFound {
					s.log.WithFields(log.Fields{
						"Module": "beaconsFQDN",
						"Data":   datum,
					}).Error(err)
				}
				continue
			}

			totalQuery := maxBeaconQuery
			s.summarizedCallback(update{
				datum.BSONKey(),
				totalQuery,
			})
		}
		s.summaryWg.Done()
	}()
}

//maxFQDNBeaconQuery finds the highest scoring fqdn beacon from this import session for a particular host
func maxFQDNBeaconQuery(datum data.UniqueIP, beaconFQDNColl *mgo.Collection, chunk int) (bson.M, error) {

	var maxBeaconFQDN struct {
		Fqdn  string  `bson:"fqdn"`
		Score float64 `bson:"score"`
	}

	mbdstQuery := maxFQDNBeaconPipeline(datum, chunk)
	err := beaconFQDNColl.Pipe(mbdstQuery).One(&maxBeaconFQDN)
	if err != nil {
		return bson.M{}, err
	}

	return bson.M{
		"$push": bson.M{
			"dat": bson.M{
				"$each": []bson.M{{
					"mbfqdn":                maxBeaconFQDN.Fqdn,
					"max_beacon_fqdn_score": maxBeaconFQDN.Score,
					"cid":                   chunk,
				}},
			},
		},
	}, nil
}

func maxFQDNBeaconPipeline(host data.UniqueIP, chunk int) []bson.M {
	return []bson.M{
		{"$match": bson.M{
			"src":              host.IP,
			"src_network_uuid": host.NetworkUUID,
			"cid":              chunk,
		}},
		// drop unnecessary data
		{"$project": bson.M{
			"fqdn":  1,
			"score": 1,
		}},
		// find the peer with the maximum score
		{"$sort": bson.M{
			"score": -1,
		}},
		{"$limit": 1},
	}
}
