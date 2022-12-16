package database

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (
	// BulkChange represents mgo upserts, updates, and removals
	BulkChange struct {
		Selector  interface{} // The selector document
		Update    interface{} // The update document if updating the document
		Upsert    bool        // Whether to insert in case the document isn't found
		Remove    bool        // Whether to remove the document found rather than updating
		SelectAll bool        // Whether to use RemoveAll/ UpdateAll
	}

	// BulkChanges is a map of collections to the changes that should be applied to each one
	BulkChanges map[string][]BulkChange

	// MgoBulkWriter is a pipeline worker which properly batches bulk updates for MongoDB
	MgoBulkWriter struct {
		db           *DB              // provides access to MongoDB
		conf         *config.Config   // contains details needed to access MongoDB
		log          *log.Logger      // main logger for RITA
		writeChannel chan BulkChanges // holds analyzed data
		writeWg      *sync.WaitGroup  // wait for writing to finish
		writerName   string           // used in error reporting
		unordered    bool             // if the operations can be applied in any order, MongoDB can run the updates in parallel
		maxBulkCount int              // max number of changes to include in each bulk update
		maxBulkSize  int              // max total size of BSON documents making up each bulk update
	}
)

// Size serializes the changes to BSON using provided buffer and returns total size
// of the BSON description of the changes. Note this method slightly underestimates the
// total amount BSON needed to describe the changes since extra flags may be sent along.
func (m BulkChange) Size(buffer []byte) ([]byte, int) {
	size := 0
	if len(buffer) > 0 { // in case the byte slice has something in it already
		buffer = buffer[:0]
	}

	if m.Selector != nil {
		buffer, _ = bson.MarshalBuffer(m.Selector, buffer)
		size += len(buffer)
		buffer = buffer[:0]
	}
	if m.Update != nil {
		buffer, _ = bson.MarshalBuffer(m.Update, buffer)
		size += len(buffer)
		buffer = buffer[:0]
	}
	return buffer, size
}

// Apply adds the change described to a bulk buffer
func (m BulkChange) Apply(bulk *mgo.Bulk) {
	if m.Selector == nil {
		return // can't describe a change without a selector
	}

	if m.Remove && m.SelectAll {
		bulk.RemoveAll(m.Selector)
	} else if m.Remove /*&& !m.SelectAll*/ {
		bulk.Remove(m.Selector)
	} else if m.Update != nil && m.Upsert {
		bulk.Upsert(m.Selector, m.Update)
	} else if m.Update != nil && m.SelectAll {
		bulk.UpdateAll(m.Selector, m.Update)
	} else if m.Update != nil /*&& !m.Upsert && !m.SelectAll*/ {
		bulk.Update(m.Selector, m.Update)
	}
}

// NewBulkWriter creates a new writer object to write output data to collections
func NewBulkWriter(db *DB, conf *config.Config, log *log.Logger, unorderedWritesOK bool, writerName string) *MgoBulkWriter {
	return &MgoBulkWriter{
		db:           db,
		conf:         conf,
		log:          log,
		writeChannel: make(chan BulkChanges),
		writeWg:      new(sync.WaitGroup),
		writerName:   writerName,
		unordered:    unorderedWritesOK,
		// Cap the bulk buffers at 500 updates. 1000 should theoretically work, but we've run into issues in the past, so we halved it.
		maxBulkCount: 500,
		// Cap the bulk buffers at 15MB. This cap ensures that our bulk transactions don't exceed the 16MB limit imposed on MongoDB docs/ operations.
		maxBulkSize: 15 * 1000 * 1000,
	}
}

// Collect sends a group of results to the writer for writing out to the database
func (w *MgoBulkWriter) Collect(data BulkChanges) {
	w.writeChannel <- data
}

// close waits for the write threads to finish
func (w *MgoBulkWriter) Close() {
	close(w.writeChannel)
	w.writeWg.Wait()
}

// start kicks off a new write thread
func (w *MgoBulkWriter) Start() {
	w.writeWg.Add(1)
	go func() {
		ssn := w.db.Session.Copy()
		defer ssn.Close()

		bulkBuffers := map[string]*mgo.Bulk{} // stores a mgo.Bulk buffer for each collection
		bulkBufferSizes := map[string]int{}   // stores the size in bytes of the BSON documents in each mgo.Bulk buffer
		bulkBufferLengths := map[string]int{} // stores the number of changes stored in each mgo.Bulk buffer
		var sizeBuffer []byte                 // used (and re-used) for BSON serialization in order to calculate the size of each BSON doc
		var changeSize int                    // holds the total size of each BSON serialized change before being added to bulkBufferSizes

		for data := range w.writeChannel { // process data as it streams into the writer
			for tgtColl, bulkChanges := range data { // loop through each collection that needs updated

				// initialize/ grab the bulk buffer associated with this collection
				bulkBuffer, bufferExists := bulkBuffers[tgtColl]
				if !bufferExists {
					bulkBuffer = ssn.DB(w.db.GetSelectedDB()).C(tgtColl).Bulk()
					if w.unordered {
						// if the order in which the updates occur doesn't matter, allow MongoDB to apply the updates in parallel
						bulkBuffer.Unordered()
					}
					bulkBuffers[tgtColl] = bulkBuffer
				}

				for _, change := range bulkChanges { // loop through each change that needs to be applied to the collection
					sizeBuffer, changeSize = change.Size(sizeBuffer)

					// if the bulk buffer has already reached the max number of changes or
					// if the total size of the bulk buffer would exceed the max size after inserting the current change
					// run the existing bulk buffer against MongoDB
					if bulkBufferLengths[tgtColl] >= w.maxBulkCount || bulkBufferSizes[tgtColl]+changeSize >= w.maxBulkSize {
						info, err := bulkBuffer.Run()
						if err != nil {
							w.log.WithFields(log.Fields{
								"Module":     w.writerName,
								"Collection": tgtColl,
								"Info":       info,
							}).Error(err)
						}
						// make sure to reset the stats we are tracking about the bulk buffer
						bulkBufferLengths[tgtColl] = 0
						bulkBufferSizes[tgtColl] = 0
					}

					// insert the change into the bulk buffer and update the stats we are tracking about the bulk buffer
					change.Apply(bulkBuffer)
					bulkBufferLengths[tgtColl]++
					bulkBufferSizes[tgtColl] += changeSize
				}
			}
		}

		// after the writer is done receiving inputs, make sure to drain all of the buffers before exiting
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
			bulkBufferSizes[tgtColl] = 0
		}
		w.writeWg.Done()
	}()
}
