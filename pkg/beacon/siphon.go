package beacon

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (

	// siphon provides a worker for making certain updates to MongoDB before the analysis phase (Evaporation)
	// this is generally for removing/updating documents that should not be analyzed or need fixing up before analysis
	// it can also pass data through to the next stage and optionally skip evaporation (Drainage)
	siphon struct {
		connLimit         int64                      // limit for strobe classification
		chunk             int                        // current chunk (0 if not on rolling analysis)
		db                *database.DB               // provides access to MongoDB
		conf              *config.Config             // contains details needed to access MongoDB
		log               *log.Logger                // main logger for RITA
		evaporateCallback func(database.BulkChanges) // operations to update/remove a uconn prior to analysis are sent to this callback
		drainCallback     func(*uconn.Input)         // gathered unique connection details are sent to this callback
		closedCallback    func()                     // called when .close() is called and no more calls to siphonCallback will be made
		siphonChannel     chan *uconn.Input          // holds dissected data
		siphonWg          sync.WaitGroup             // wait for writing to finish
	}
)

// newSiphon creates a new siphon for beacon data
func newSiphon(connLimit int64, chunk int, db *database.DB, conf *config.Config, log *log.Logger, evaporateCallback func(database.BulkChanges), drainCallback func(*uconn.Input), closedCallback func()) *siphon {
	return &siphon{
		connLimit:         connLimit,
		chunk:             chunk,
		db:                db,
		conf:              conf,
		log:               log,
		evaporateCallback: evaporateCallback,
		drainCallback:     drainCallback,
		closedCallback:    closedCallback,
		siphonChannel:     make(chan *uconn.Input),
	}
}

// collect sends a group of results to the siphon for optionally updating in the database
func (s *siphon) collect(data *uconn.Input) {
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
				// if uconn became a strobe just from the current chunk, then we would not have received it here
				// as uconns upgrades itself to a strobe if its connection count met the strobe thresh for this chunk only
				// and the dissector filters out strobes

				// if uconn became a strobe during this chunk over its cummulative connection count over all chunks,
				// then we must upgrade it to a strobe and remove the timestamp and bytes arrays from the current chunk
				// or else the uconn document can grow to unacceptable sizes
				// these tasks are to be handled prior to sorting & analysis
				actions := database.BulkChanges{
					s.conf.T.Structure.UniqueConnTable: []database.BulkChange{{
						Selector: database.MergeBSONMaps(data.Hosts.BSONKey(), bson.M{
							"dat": bson.M{"$elemMatch": bson.M{
								"cid":   s.chunk,
								"ts":    bson.M{"$exists": true},
								"bytes": bson.M{"$exists": true},
							}},
						}),
						Update: bson.M{
							// set the uconn as a strobe
							// this must be done as uconns unsets its strobe flag if the current chunk doesnt meet
							// the strobe limit
							"$set": bson.M{"strobe": true},
							// remove the bytes and ts arrays for the current chunk in the uconn document
							"$unset": bson.M{"dat.$.ts": "", "dat.$.bytes": ""},
						},
					}},

					// remove the uconn from the beacon table as its now a strobe
					s.conf.T.Beacon.BeaconTable: []database.BulkChange{{
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
