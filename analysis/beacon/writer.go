package beacon

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/datatypes/beacon"
)

type (
	//writer simply writes AnalysisOutput objects to the beacons collection
	writer struct {
		db           *database.DB                // provides access to MongoDB
		conf         *config.Config              // contains details needed to access MongoDB
		writeChannel chan *beacon.AnalysisOutput // holds analyzed data
		writeWg      sync.WaitGroup              // wait for writing to finish
	}
)

//newWriter creates a writer object to write AnalysisOutput data to
//the beacons collection
func newWriter(db *database.DB, conf *config.Config) *writer {
	return &writer{
		db:           db,
		conf:         conf,
		writeChannel: make(chan *beacon.AnalysisOutput),
	}
}

//write queues up a AnalysisOutput to be written to the beacons collection
//Note: this function may block
func (w *writer) write(data *beacon.AnalysisOutput) {
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

		//TODO: Implement bulk writes
		for data := range w.writeChannel {
			ssn.DB(w.db.GetSelectedDB()).C(w.conf.T.Beacon.BeaconTable).Insert(data)

		}
		w.writeWg.Done()
	}()
}
