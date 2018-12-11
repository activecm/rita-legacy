package blacklist

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
)

type (
	//writer writes output structure objects to blacklisted collections
	writer struct {
		targetCollection string
		db               *database.DB     // provides access to MongoDB
		conf             *config.Config   // contains details needed to access MongoDB
		writeChannel     chan interface{} // holds analyzed data
		writeWg          sync.WaitGroup   // wait for writing to finish
	}
)

//newWriter creates a writer object to write output data to
//the beacons collection
func newWriter(targetCollection string, db *database.DB, conf *config.Config) *writer {
	return &writer{
		targetCollection: targetCollection,
		db:               db,
		conf:             conf,
		writeChannel:     make(chan interface{}),
	}
}

//write queues up an output structure to be written to a blacklisted collection
func (w *writer) write(data interface{}) {
	w.writeChannel <- data
}

// close waits for the write threads to finish
func (w *writer) close() {
	close(w.writeChannel)
	w.writeWg.Wait()
}

// start kicks off a new write thread
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
