package uconnproxy

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	log "github.com/sirupsen/logrus"
)

type (
	//writer blah blah
	writer struct { //structure for writing proxy beacons  results to mongo
		targetCollection string
		db               *database.DB   // provides access to MongoDB
		conf             *config.Config // contains details needed to access MongoDB
		log              *log.Logger    // main logger for RITA
		writeChannel     chan update    // holds analyzed data
		writeWg          sync.WaitGroup // wait for writing to finish
	}
)

//newWriter creates a new writer object to write output data to proxy beacons collections
func newWriter(targetCollection string, db *database.DB, conf *config.Config, log *log.Logger) *writer {
	return &writer{
		targetCollection: targetCollection,
		db:               db,
		conf:             conf,
		log:              log,
		writeChannel:     make(chan update),
	}
}

//collect sends a group of results to the writer for writing out to the database
func (w *writer) collect(data update) {
	w.writeChannel <- data
}

//close waits for the write threads to finish
func (w *writer) close() {
	close(w.writeChannel)
	w.writeWg.Wait()
}

//start kicks off a new write thread
func (w *writer) start() {
	w.writeWg.Add(1)
	go func() {
		ssn := w.db.Session.Copy()
		defer ssn.Close()

		for data := range w.writeChannel {

			if data.uconnProxy.query != nil {

				info, err := ssn.DB(w.db.GetSelectedDB()).C(w.targetCollection).Upsert(data.uconnProxy.selector, data.uconnProxy.query)

				if err != nil ||
					((info.Updated == 0) && (info.UpsertedId == nil)) {
					w.log.WithFields(log.Fields{
						"Module": "uconnsproxy",
						"Info":   info,
						"Data":   data,
					}).Error(err)
				}
			}
		}
		w.writeWg.Done()
	}()
}
