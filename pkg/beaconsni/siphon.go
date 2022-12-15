package beaconsni

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (

	// siphon provides a worker for making certain updates to MongoDB before the analysis phase (Evaporation)
	// this is generally for removing/updating documents that should not be analyzed or need fixing up before analysis
	// it can also pass data through to the next stage and optionally skip evaporation (Drainage)
	siphon struct {
		connLimit         int64                      // limit for strobe classification
		chunk             int                        //current chunk (0 if not on rolling analysis)
		db                *database.DB               // provides access to MongoDB
		conf              *config.Config             // contains details needed to access MongoDB
		log               *log.Logger                // main logger for RITA
		evaporateCallback func(database.BulkChanges) // operations to update/remove a sniconn prior to analysis are sent to this callback
		drainCallback     func(*dissectorResults)    // gathered unique connection details are sent to this callback
		closedCallback    func()                     // called when .close() is called and no more calls to siphonCallback will be made
		siphonChannel     chan *dissectorResults     // holds dissected data
		siphonWg          sync.WaitGroup             // wait for writing to finish
	}
)

// newSiphon creates a new siphon for beacon data
func newSiphon(connLimit int64, chunk int, db *database.DB, conf *config.Config, log *log.Logger, evaporateCallback func(database.BulkChanges), drainCallback func(*dissectorResults), closedCallback func()) *siphon {
	return &siphon{
		connLimit:         connLimit,
		chunk:             chunk,
		db:                db,
		conf:              conf,
		log:               log,
		evaporateCallback: evaporateCallback,
		drainCallback:     drainCallback,
		closedCallback:    closedCallback,
		siphonChannel:     make(chan *dissectorResults),
	}
}

// collect sends a group of results to the siphon for optionally updating in the database
func (s *siphon) collect(data *dissectorResults) {
	s.siphonChannel <- data
}

// close waits for the siphon threads to finish
func (s *siphon) close() {
	close(s.siphonChannel)
	s.siphonWg.Wait()
	s.closedCallback()
}

// start kicks off a new siphon thread
func (s *siphon) start() {
	s.siphonWg.Add(1)
	go func() {
		ssn := s.db.Session.Copy()
		defer ssn.Close()

		for data := range s.siphonChannel {

			// check if uconn has become a strobe
			if data.ConnectionCount > s.connLimit {
				// if sniconn became a strobe just from the current chunk, then we would not have received it here
				// as sniconn upgrades itself to a strobe if either of its tls or http connection counts met the strobe
				// thresh for this chunk only

				// if sniconn became a strobe during this chunk over its cumulative connection count over all chunks,
				// then we must upgrade it to a strobe and remove the timestamp and bytes arrays from the current chunk
				// or else the sniconn document can grow to unacceptable sizes
				// these tasks are to be handled prior to sorting & analysis
				pairSelector := data.Hosts.BSONKey()

				actions := database.BulkChanges{
					s.conf.T.Structure.SNIConnTable: []database.BulkChange{
						{ // remove the bytes and ts arrays for both tls & http in the current chunk in the sniconn document
							Selector: database.MergeBSONMaps(pairSelector, bson.M{
								"dat": bson.M{"$elemMatch": bson.M{
									"cid": s.chunk,
									"tls": bson.M{"$exists": true},
								}},
							}),
							Update: bson.M{
								"$unset": bson.M{
									"dat.$.tls.bytes": "",
									"dat.$.tls.ts":    "",
								},
							},
						},
						{ // this update has to be done separately since $ only matches one subdocument at a time
							Selector: database.MergeBSONMaps(pairSelector, bson.M{
								"dat": bson.M{"$elemMatch": bson.M{
									"cid":  s.chunk,
									"http": bson.M{"$exists": true},
								}},
							}),
							Update: bson.M{
								"$unset": bson.M{
									"dat.$.http.bytes": "",
									"dat.$.http.ts":    "",
								},
							},
						},
						{ // set the sniconn as a strobe via the merged property.
							// This is being done here and not in SNIconns because beaconSNI merges the connections from
							// multiple protocols together whereas SNIconn tracks the strobe statuses separately
							Selector: pairSelector,
							Update: bson.M{"$push": bson.M{
								"dat": bson.M{"$each": []bson.M{{
									"cid": s.chunk,
									"merged": bson.M{
										"strobe": true,
									},
								}}},
							}},
							Upsert: true,
						},
					},

					// remove the sniconn from the beaconsni table as its now a strobe
					s.conf.T.BeaconSNI.BeaconSNITable: []database.BulkChange{{
						Selector: data.Hosts.BSONKey(),
						Remove:   true,
					}},
				}
				// evaporate uconn via the bulk writer
				s.evaporateCallback(actions)
			} else {
				// if uconn is not a strobe, drain it down into the rest of the
				// beacon analysis pipeline
				s.drainCallback(data)
			}
		}
		s.siphonWg.Done()
	}()
}
