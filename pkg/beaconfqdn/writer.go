package beaconfqdn

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
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

		var count int
		var sizeBuffer []byte
		var sizeBytes int64

		for data := range w.writeChannel {

			// we keep track of record sizes since the documents coming out of the
			// FQDN analysis can get rather larger (see the ds.bytes field in particular)
			var actionSize int64

			sizeBuffer, _ = bson.MarshalBuffer(data.selector, sizeBuffer)
			actionSize = actionSize + int64(len(sizeBuffer))
			sizeBuffer = sizeBuffer[:0]

			sizeBuffer, _ = bson.MarshalBuffer(data.query, sizeBuffer)
			actionSize = actionSize + int64(len(sizeBuffer))
			sizeBuffer = sizeBuffer[:0]

			// Break up bulk writes such that each batch is at most 1000 documents
			// and is smaller than 16MB. We use 15MB here instead of 16 since
			// there is document size overhead beyond just the selectors and queries.
			if count+1 == 501 || sizeBytes+actionSize >= 15*1000*1000 {
				info, err := bulk.Run()
				if err != nil {
					w.log.WithFields(log.Fields{
						"Module": "beaconsFQDN",
						"Info":   info,
					}).Error(err)
				}
				count = 0
				sizeBytes = 0
			}

			bulk.Upsert(data.selector, data.query)
			count++
			sizeBytes += actionSize
		}

		info, err := bulk.Run()
		if err != nil {
			w.log.WithFields(log.Fields{
				"Module": "beaconsFQDN",
				"Info":   info,
			}).Error(err)
		}
		// count = 0
		// sizeBytes = 0
		w.writeWg.Done()
	}()
}
