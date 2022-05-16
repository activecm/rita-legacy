package beaconproxy

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo"
	log "github.com/sirupsen/logrus"
)

type (
	//mgoBulkWriter provides a worker for writing bulk actions to MongoDB
	mgoBulkWriter struct { //structure for writing results to mongo
		db           *database.DB        // provides access to MongoDB
		conf         *config.Config      // contains details needed to access MongoDB
		log          *log.Logger         // main logger for RITA
		writeChannel chan mgoBulkActions // holds analyzed data
		writeWg      sync.WaitGroup      // wait for writing to finish
		writerName   string              // used in error reporting
	}
)

//newMgoBulkWriter creates a new writer object to write output data to collections
func newMgoBulkWriter(db *database.DB, conf *config.Config, log *log.Logger, writerName string) *mgoBulkWriter {
	return &mgoBulkWriter{
		db:           db,
		conf:         conf,
		log:          log,
		writeChannel: make(chan mgoBulkActions),
		writerName:   writerName,
	}
}

//collect sends a group of results to the writer for writing out to the database
func (w *mgoBulkWriter) collect(data mgoBulkActions) {
	w.writeChannel <- data
}

//close waits for the write threads to finish
func (w *mgoBulkWriter) close() {
	close(w.writeChannel)
	w.writeWg.Wait()
}

//start kicks off a new write thread
func (w *mgoBulkWriter) start() {
	w.writeWg.Add(1)
	go func() {
		ssn := w.db.Session.Copy()
		defer ssn.Close()

		bulkBuffers := map[string]*mgo.Bulk{}
		bulkBufferLengths := map[string]int{}

		for data := range w.writeChannel {
			for tgtColl, bulkCallback := range data {
				bulkBuffer, bufferExists := bulkBuffers[tgtColl]
				if !bufferExists {
					bulkBuffer = ssn.DB(w.db.GetSelectedDB()).C(tgtColl).Bulk()
					bulkBuffers[tgtColl] = bulkBuffer
				}

				bulkBufferLengths[tgtColl] += bulkCallback(bulkBuffer)

				if bulkBufferLengths[tgtColl] >= 1000 {
					info, err := bulkBuffer.Run()
					if err != nil {
						w.log.WithFields(log.Fields{
							"Module":     w.writerName,
							"Collection": tgtColl,
							"Info":       info,
						}).Error(err)
					}

					bulkBufferLengths[tgtColl] = 0
				}
			}
		}
		for tgtColl, bulkBuffer := range bulkBuffers {
			info, err := bulkBuffer.Run()
			if err != nil {
				w.log.WithFields(log.Fields{
					"Module":     w.writerName,
					"Collection": tgtColl,
					"Info":       info,
				}).Error(err)
			}

			bulkBufferLengths[tgtColl] = 0
		}
		w.writeWg.Done()
	}()
}

// package beaconproxy

// import (
// 	"sync"

// 	"github.com/activecm/rita/config"
// 	"github.com/activecm/rita/database"
// 	log "github.com/sirupsen/logrus"
// )

// type (
// 	writer struct {
// 		targetCollection string
// 		db               *database.DB   // provides access to MongoDB
// 		conf             *config.Config // contains details needed to access MongoDB
// 		log              *log.Logger    // main logger for RITA
// 		writeChannel     chan *update   // holds analyzed data
// 		writeWg          sync.WaitGroup // wait for writing to finish
// 	}
// )

// //newWriter creates a new writer object to write output data to beaconproxy collections
// func newWriter(targetCollection string, db *database.DB, conf *config.Config, log *log.Logger) *writer {
// 	return &writer{
// 		targetCollection: targetCollection,
// 		db:               db,
// 		conf:             conf,
// 		log:              log,
// 		writeChannel:     make(chan *update),
// 	}
// }

// //collect sends a group of results to the writer for writing out to the database
// func (w *writer) collect(data *update) {
// 	w.writeChannel <- data
// }

// //close waits for the write threads to finish
// func (w *writer) close() {
// 	close(w.writeChannel)
// 	w.writeWg.Wait()
// }

// //start kicks off a new write thread
// func (w *writer) start() {
// 	w.writeWg.Add(1)
// 	go func() {
// 		ssn := w.db.Session.Copy()
// 		defer ssn.Close()

// 		for data := range w.writeChannel {

// 			if data.beacon.query != nil {
// 				// update beacons proxy table
// 				info, err := ssn.DB(w.db.GetSelectedDB()).C(w.targetCollection).Upsert(data.beacon.selector, data.beacon.query)

// 				if err != nil ||
// 					((info.Updated == 0) && (info.UpsertedId == nil)) {
// 					w.log.WithFields(log.Fields{
// 						"Module": "beaconsProxy",
// 						"Info":   info,
// 						"Data":   data,
// 					}).Error(err)
// 				}

// 				// update hosts table with max beacon proxy updates
// 				if data.hostBeacon.query != nil {
// 					// update hosts table
// 					info, err = ssn.DB(w.db.GetSelectedDB()).C(w.conf.T.Structure.HostTable).Upsert(data.hostBeacon.selector, data.hostBeacon.query)

// 					if err != nil ||
// 						((info.Updated == 0) && (info.UpsertedId == nil) && (info.Matched == 0)) {
// 						w.log.WithFields(log.Fields{
// 							"Module": "beaconsProxy",
// 							"Info":   info,
// 							"Data":   data,
// 						}).Error(err)
// 					}
// 				}
// 			}

// 			if data.uconnproxy.query != nil {
// 				// update uconnsproxy table
// 				info, err := ssn.DB(w.db.GetSelectedDB()).C(w.conf.T.Structure.UniqueConnProxyTable).Upsert(data.uconnproxy.selector, data.uconnproxy.query)

// 				if err != nil ||
// 					((info.Updated == 0) && (info.UpsertedId == nil)) {
// 					w.log.WithFields(log.Fields{
// 						"Module": "beaconsProxy",
// 						"Info":   info,
// 						"Data":   data,
// 					}).Error(err)
// 				}

// 				//delete the record (no longer a beacon - its a strobe)
// 				info, err = ssn.DB(w.db.GetSelectedDB()).C(w.targetCollection).RemoveAll(data.uconnproxy.selector)
// 				if err != nil ||
// 					((info.Updated == 0) && (info.Removed == 0) && (info.Matched == 0) && (info.UpsertedId == nil)) {
// 					w.log.WithFields(log.Fields{
// 						"Module": "beaconsProxy",
// 						"Info":   info,
// 						"Data":   data,
// 					}).Error(err)
// 				}
// 			}
// 		}
// 		w.writeWg.Done()
// 	}()
// }
