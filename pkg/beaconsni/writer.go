package beaconsni

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

				// limit the buffer to 500 to prevent hitting 16MB limit
				// 1000 breaks this limit, hitting 17MB at times
				if bulkBufferLengths[tgtColl] >= 500 {
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
