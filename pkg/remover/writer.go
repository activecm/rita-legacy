package beacon

import (
	"fmt"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
)

type (
	writer struct {
		cid          int            // chuck id for deletion
		db           *database.DB   // provides access to MongoDB
		conf         *config.Config // contains details needed to access MongoDB
		writeChannel chan string    // holds analyzed data
		writeWg      sync.WaitGroup // wait for writing to finish
	}
)

//newWriter creates a new writer object to write output data
func newWriter(targetCollection string, db *database.DB, conf *config.Config) *writer {
	return &writer{
		cid:          int,
		db:           db,
		conf:         conf,
		writeChannel: make(chan string),
	}
}

//collect sends a group of results to the writer for writing out to the database
func (w *writer) collect(data string) {
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

			//delete the record if it hasn't been updated since the target delete chunk
			_, err := ssn.DB(w.db.GetSelectedDB()).C(data).RemoveAll(bson.M{"cid": w.cid})
			if err != nil {
				fmt.Println(err)
			}

		}
		w.writeWg.Done()
	}()
}
