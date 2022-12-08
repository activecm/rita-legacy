package beaconproxy

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/uconnproxy"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (
	// flexible query instructions for making updates to mongo
	evaporator struct {
		collection   string     // target collection for update
		selector     bson.M     // selector for finding item to update
		query        mgo.Change // update/remove/upsert query
		arrayFilters []bson.M   // (optional) array filters when using UpdateWithArrayFilters
	}

	siphonInput struct {
		// only set one or the other unless you want to update a document before passing it onto the next stage
		Drain     *uconnproxy.Input // unique connection to pass through to the next stage
		Evaporate []evaporator      // database actions to perform on documents that need to be removed/updated... evaporated
	}

	// siphon provides a worker for making certain updates to MongoDB before the analysis phase (Evaporation)
	// this is generally for removing/updating documents that should not be analyzed or need fixing up before analysis
	// it can also pass data through to the next stage and optionally skip evaporation (Drainage)
	siphon struct {
		db             *database.DB            // provides access to MongoDB
		conf           *config.Config          // contains details needed to access MongoDB
		log            *log.Logger             // main logger for RITA
		siphonCallback func(*uconnproxy.Input) // gathered unique connection details are sent to this callback
		closedCallback func()                  // called when .close() is called and no more calls to siphonCallback will be made
		siphonChannel  chan siphonInput        // holds dissected data
		siphonWg       sync.WaitGroup          // wait for writing to finish
	}
)

// newSiphon creates a new siphon for beacon data
func newSiphon(db *database.DB, conf *config.Config, log *log.Logger, siphonCallback func(*uconnproxy.Input), closedCallback func()) *siphon {
	return &siphon{
		db:             db,
		conf:           conf,
		log:            log,
		siphonCallback: siphonCallback,
		closedCallback: closedCallback,
		siphonChannel:  make(chan siphonInput),
	}
}

// collect sends a group of results to the siphon for optionally updating in the database
func (s *siphon) collect(data siphonInput) {
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
			// if there are evaporation tasks to complete, run through all of them with normal (non-bulk) queries
			// because mgo does not support updating with array filters in bulk operations
			if data.Evaporate != nil {
				for _, action := range data.Evaporate {
					// run UpdateWithArrayFilters if arrayFilters are set
					if action.arrayFilters != nil {
						info, err := ssn.DB(s.db.GetSelectedDB()).C(action.collection).UpdateWithArrayFilters(
							action.selector, action.query.Update, action.arrayFilters, true)
						if err != nil ||
							((info.Updated == 0) && (info.Removed == 0) && (info.Matched == 0) && (info.UpsertedId == nil)) {
							s.log.WithFields(log.Fields{
								"Module":     "beacon",
								"Collection": action.collection,
								"Info":       info,
								"Data":       data,
							}).Error(err.Error())
						}
					} else {
						// run query.Apply() for all other actions since it is pretty flexible for different uses
						info, err := ssn.DB(s.db.GetSelectedDB()).C(action.collection).Find(action.selector).Apply(action.query, nil)
						if err != nil ||
							((info.Updated == 0) && (info.Removed == 0) && (info.Matched == 0) && (info.UpsertedId == nil)) {
							s.log.WithFields(log.Fields{
								"Module":     "beacon",
								"Collection": action.collection,
								"Info":       info,
								"Data":       data,
							}).Error(err.Error())
						}
					}
				}
				// if a drain is specified, drain it down to the next stage
			} else if data.Drain != nil {
				s.siphonCallback(data.Drain)
			}
		}
		s.siphonWg.Done()
	}()
}
