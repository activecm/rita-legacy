package beacon

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	dataBeacon "github.com/activecm/rita/datatypes/beacon"
)

type (
	writer struct {
		db           *database.DB
		conf         *config.Config
		writeChannel chan *dataBeacon.BeaconAnalysisOutput // holds analyzed data
		writeWg      sync.WaitGroup                        // wait for writing to finish
	}
)

func newWriter(db *database.DB, conf *config.Config) *writer {
	return &writer{
		db:           db,
		conf:         conf,
		writeChannel: make(chan *dataBeacon.BeaconAnalysisOutput),
	}
}

func (w *writer) write(data *dataBeacon.BeaconAnalysisOutput) {
	w.writeChannel <- data
}

func (w *writer) flush() {
	close(w.writeChannel)
	w.writeWg.Wait()
}

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
