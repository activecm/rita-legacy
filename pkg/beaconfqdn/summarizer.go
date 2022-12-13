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

// newSummarizer creates a new summarizer for fqdn beacon data
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

// collect collects an internal host to create summary data for
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
			beaconFQDNCollection := ssn.DB(s.db.GetSelectedDB()).C(s.conf.T.BeaconFQDN.BeaconFQDNTable)
			hostCollection := ssn.DB(s.db.GetSelectedDB()).C(s.conf.T.Structure.HostTable)

			maxFQDNBeaconSelector, maxFQDNBeaconQuery, err := maxFQDNBeaconUpdate(
				datum, beaconFQDNCollection, hostCollection, s.chunk,
			)
			if err != nil {
				if err != mgo.ErrNotFound {
					s.log.WithFields(log.Fields{
						"Module": "beaconsFQDN",
						"Data":   datum,
					}).Error(err)
				}
				continue
			}

			if len(maxFQDNBeaconQuery) > 0 {
				s.summarizedCallback(update{
					maxFQDNBeaconSelector,
					maxFQDNBeaconQuery,
				})
			}
		}
		s.summaryWg.Done()
	}()
}

// maxFQDNBeaconUpdate finds the highest scoring fqdn beacon from this import session for a particular host
func maxFQDNBeaconUpdate(datum data.UniqueIP, beaconFQDNColl, hostColl *mgo.Collection, chunk int) (bson.M, bson.M, error) {

	var maxBeaconFQDN struct {
		Fqdn  string  `bson:"fqdn"`
		Score float64 `bson:"score"`
	}

	mbdstQuery := maxFQDNBeaconPipeline(datum, chunk)
	err := beaconFQDNColl.Pipe(mbdstQuery).One(&maxBeaconFQDN)
	if err != nil {
		return nil, nil, err
	}

	hostSelector := datum.BSONKey()
	hostWithDatEntrySelector := database.MergeBSONMaps(
		hostSelector,
		bson.M{"dat.mbfqdn": maxBeaconFQDN.Fqdn},
	)

	nExistingEntries, err := hostColl.Find(hostWithDatEntrySelector).Count()
	if err != nil {
		return nil, nil, err
	}

	if nExistingEntries > 0 {
		// just need to update the cid and score if there is an
		// an existing record
		updateQuery := bson.M{
			"$set": bson.M{
				"dat.$.max_beacon_fqdn_score": maxBeaconFQDN.Score,
				"dat.$.cid":                   chunk,
			},
		}
		return hostWithDatEntrySelector, updateQuery, nil
	}

	insertQuery := bson.M{
		"$push": bson.M{
			"dat": bson.M{
				"$each": []bson.M{{
					"mbfqdn":                maxBeaconFQDN.Fqdn,
					"max_beacon_fqdn_score": maxBeaconFQDN.Score,
					"cid":                   chunk,
				}},
			},
		},
	}

	return hostSelector, insertQuery, nil
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
