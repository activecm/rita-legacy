package blacklist

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
)

type (
	writer struct { //structure for writing blacklist results to mongo
		targetCollection string
		db               *database.DB     // provides access to MongoDB
		conf             *config.Config   // contains details needed to access MongoDB
		writeChannel     chan interface{} // holds analyzed data
		writeWg          sync.WaitGroup   // wait for writing to finish
	}
)

//newWriter creates a new writer object to write output data to blacklisted collections
func newWriter(targetCollection string, db *database.DB, conf *config.Config) *writer {
	return &writer{
		targetCollection: targetCollection,
		db:               db,
		conf:             conf,
		writeChannel:     make(chan interface{}),
	}
}

//write sends a group of results to the writer for writing out to the database
func (w *writer) write(data interface{}) {
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
			ssn.DB(w.db.GetSelectedDB()).C(w.targetCollection).Insert(data)
		}
		w.writeWg.Done()
	}()
}
