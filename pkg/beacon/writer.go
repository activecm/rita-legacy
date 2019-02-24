package beacon

import (
	"fmt"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
)

type (
	writer struct {
		targetCollection string
		db               *database.DB   // provides access to MongoDB
		conf             *config.Config // contains details needed to access MongoDB
		writeChannel     chan *update   // holds analyzed data
		writeWg          sync.WaitGroup // wait for writing to finish
	}
)

//newWriter creates a new writer object to write output data to blacklisted collections
func newWriter(targetCollection string, db *database.DB, conf *config.Config) *writer {
	return &writer{
		targetCollection: targetCollection,
		db:               db,
		conf:             conf,
		writeChannel:     make(chan *update),
	}
}

//collect sends a group of results to the writer for writing out to the database
func (w *writer) collect(data *update) {
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

			if data.beacon.query != nil {
				// update beacons table
				info, err := ssn.DB(w.db.GetSelectedDB()).C(w.targetCollection).Upsert(data.beacon.selector, data.beacon.query)

				if err != nil ||
					((info.Updated == 0) && (info.UpsertedId == nil)) {
					fmt.Println(err, info, data)
				}

				// update hosts table
				info, err = ssn.DB(w.db.GetSelectedDB()).C(w.conf.T.Structure.HostTable).Upsert(data.host.selector, data.host.query)

				if err != nil ||
					((info.Updated == 0) && (info.UpsertedId == nil) && (info.Matched == 0)) {
					fmt.Println(err, info, data)
				}
			}

			if data.uconn.query != nil {
				// update uconns table
				info, err := ssn.DB(w.db.GetSelectedDB()).C(w.conf.T.Structure.UniqueConnTable).Upsert(data.uconn.selector, data.uconn.query)

				if err != nil ||
					((info.Updated == 0) && (info.UpsertedId == nil)) {
					fmt.Println(err, info, data)
				}

				//delete the record (no longer a beacon - its a strobe)
				info, err = ssn.DB(w.db.GetSelectedDB()).C(w.targetCollection).RemoveAll(data.uconn.selector)
				if err != nil ||
					((info.Updated == 0) && (info.Removed == 0) && (info.Matched == 0) && (info.UpsertedId == nil)) {
					fmt.Println(err, info, data)
				}
			}

		}
		w.writeWg.Done()
	}()
}
