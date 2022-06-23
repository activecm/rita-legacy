package host

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	log "github.com/sirupsen/logrus"
)

type (
	//writer provides a worker for writing bulk upserts to MongoDB
	writer struct { //structure for writing results to mongo
		targetCollection string
		db               *database.DB   // provides access to MongoDB
		conf             *config.Config // contains details needed to access MongoDB
		log              *log.Logger    // main logger for RITA
		writeChannel     chan update    // holds analyzed data
		writeWg          sync.WaitGroup // wait for writing to finish
	}
)

//newWriter creates a new writer object to write output data to collections
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

		bulk := ssn.DB(w.db.GetSelectedDB()).C(w.targetCollection).Bulk()
		bulk.Unordered()
		count := 0

		for data := range w.writeChannel {
			bulk.Upsert(data.selector, data.query)
			count++

			// limit the buffer to 500 to prevent hitting 16MB limit
			// 1000 breaks this limit, hitting 17MB at times
			if count >= 500 {
				info, err := bulk.Run()
				if err != nil {
					w.log.WithFields(log.Fields{
						"Module": "host",
						"Info":   info,
					}).Error(err)
				}
				count = 0
			}
		}

		info, err := bulk.Run()
		if err != nil {
			w.log.WithFields(log.Fields{
				"Module": "host",
				"Info":   info,
			}).Error(err)
		}
		// count = 0
		w.writeWg.Done()
	}()
}
