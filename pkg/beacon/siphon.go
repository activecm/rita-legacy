package beacon

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (
	evaporator struct {
		collection   string
		selector     bson.M
		query        mgo.Change
		arrayFilters []bson.M
	}

	siphonInput struct {
		Drain     *uconn.Input
		Evaporate []evaporator
	}

	// siphon provides a worker for making certain updates to MongoDB and passing data through to the next stage
	siphon struct {
		db             *database.DB       // provides access to MongoDB
		conf           *config.Config     // contains details needed to access MongoDB
		log            *log.Logger        // main logger for RITA
		siphonCallback func(*uconn.Input) // gathered unique connection details are sent to this callback
		closedCallback func()             // called when .close() is called and no more calls to siphonCallback will be made
		writeChannel   chan siphonInput   // holds analyzed data
		writeWg        sync.WaitGroup     // wait for writing to finish
	}
)

// newSiphon creates a new siphon for beacon data
func newSiphon(db *database.DB, conf *config.Config, log *log.Logger, siphonCallback func(*uconn.Input), closedCallback func()) *siphon {
	return &siphon{
		db:             db,
		conf:           conf,
		log:            log,
		siphonCallback: siphonCallback,
		closedCallback: closedCallback,
		writeChannel:   make(chan siphonInput),
	}
}

// collect sends a group of results to the writer for writing out to the database
func (w *siphon) collect(data siphonInput) {
	w.writeChannel <- data
}

// close waits for the write threads to finish
func (w *siphon) close() {
	close(w.writeChannel)
	w.writeWg.Wait()
	w.closedCallback()
}

// start kicks off a new write thread
func (w *siphon) start() {
	w.writeWg.Add(1)
	go func() {
		ssn := w.db.Session.Copy()
		defer ssn.Close()

		for data := range w.writeChannel {
			if data.Evaporate != nil {
				for _, action := range data.Evaporate {
					if action.arrayFilters != nil {
						info, err := ssn.DB(w.db.GetSelectedDB()).C(action.collection).UpdateWithArrayFilters(action.selector, action.query.Update, action.arrayFilters, true)
						if err != nil ||
							((info.Updated == 0) && (info.Removed == 0) && (info.Matched == 0) && (info.UpsertedId == nil)) {
							w.log.WithFields(log.Fields{
								"Module":     "beacon",
								"Collection": action.collection,
								"Info":       info,
								"Data":       data,
							}).Error(err.Error())
						}
					} else {
						info, err := ssn.DB(w.db.GetSelectedDB()).C(action.collection).Find(action.selector).Apply(action.query, nil)
						if err != nil ||
							((info.Updated == 0) && (info.Removed == 0) && (info.Matched == 0) && (info.UpsertedId == nil)) {
							w.log.WithFields(log.Fields{
								"Module":     "beacon",
								"Collection": action.collection,
								"Info":       info,
								"Data":       data,
							}).Error(err.Error())
						}
					}
				}
			} else if data.Drain != nil {
				w.siphonCallback(data.Drain)
			}
		}
		w.writeWg.Done()
	}()
}
