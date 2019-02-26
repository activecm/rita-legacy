package certificate

import (
	"fmt"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
)

type (
	//writer blah blah
	writer struct { //structure for writing blacklist results to mongo
		db           *database.DB   // provides access to MongoDB
		conf         *config.Config // contains details needed to access MongoDB
		writeChannel chan update    // holds analyzed data
		writeWg      sync.WaitGroup // wait for writing to finish
	}
)

//newWriter creates a new writer object to write output data to blacklisted collections
func newWriter(db *database.DB, conf *config.Config) *writer {
	return &writer{
		db:           db,
		conf:         conf,
		writeChannel: make(chan update),
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

			info, err := ssn.DB(w.db.GetSelectedDB()).C(data.collection).Upsert(data.selector, data.query)
			if err != nil ||
				((info.Updated == 0) && (info.UpsertedId == nil)) {
				fmt.Println(err, info, data)
			}

		}
		w.writeWg.Done()
	}()
}
