package parser

import (
	"sync"

	"github.com/bglebrun/rita/database"

	log "github.com/Sirupsen/logrus"
)

type (
	// Document holds one item to be written to a database
	WriteQueuedLine struct {
		line ParsedLine            // Thing to write
		file *database.IndexedFile // The file which it came from
	}

	// DocWriter writes documents to a database
	DocWriter struct {
		res          *database.Resources   // Handle to the app resources
		importWl     bool                  // Flag to import whitelist
		whitelist    []string              // Pointer to our whitelist array
		writeChannel chan *WriteQueuedLine // Write channel
		writeWG      *sync.WaitGroup       // Used to block until complete
		databases    []string              // Track the db states, cached
		dblock       *sync.Mutex           // For the databases fields
		started      bool                  // Track if we've started the writer
		threadCount  int                   // Number of write threads
	}
)

// New generates a new DocWriter
func NewDocWriter(res *database.Resources, threadCount int) *DocWriter {
	return &DocWriter{
		res:          res,
		importWl:     res.System.ImportWhitelist,
		whitelist:    res.System.Whitelist,
		writeChannel: make(chan *WriteQueuedLine, 5000),
		writeWG:      new(sync.WaitGroup),
		dblock:       new(sync.Mutex),
		started:      false,
		threadCount:  threadCount,
	}
}

// Start begins the DocWriter spinning on its input
func (d *DocWriter) Start() {
	// Add a second layer of protection against untracked starts.
	if !d.started {
		for i := 0; i < d.threadCount; i++ {
			d.started = true
			go d.writeLoop()
		}
	}
	return
}

// Write allows a user to add to the channel
func (d *DocWriter) Write(doc *WriteQueuedLine) {
	d.writeChannel <- doc
	return
}

// Flush writes the final documents to the db and exits docwriter
func (d *DocWriter) Flush() {
	d.res.Log.Debug("closing write channel")
	close(d.writeChannel)
	d.res.Log.Debug("waiting for writes to finish")
	d.writeWG.Wait()
	d.res.Log.Debug("writes completed, exiting write loop")
	return
}

// writeLoop loops over the input channel spawning threads to write
func (d *DocWriter) writeLoop() {
	var err error

	//Add 1 to wait group to signify we are writing (for flush)
	d.writeWG.Add(1)

	ssn := d.res.DB.Session.Copy()
	defer ssn.Close()

	for {
		//Get next doc to write
		doc, ok := <-d.writeChannel
		if !ok {
			d.res.Log.Info("Exiting write loop.")
			break
		}

		//adjust for dates
		targetDB := doc.file.Database
		if d.res.System.BroConfig.UseDates {
			targetDB += "-" + doc.file.Date
		}

		//check if we need to add this database
		seen := false
		d.dblock.Lock()
		for _, aval := range d.databases {
			if aval == targetDB {
				seen = true
				break
			}
		}

		if !seen {
			d.res.MetaDB.AddNewDB(targetDB)
			d.databases = append(d.databases, targetDB)
		}
		d.dblock.Unlock()

		//If the doc isn't whitelisted, insert it
		if !(d.importWl && doc.line.IsWhiteListed(d.whitelist)) {
			err = ssn.DB(targetDB).C(doc.line.TargetCollection()).Insert(doc.line)
		}

		if err != nil {
			d.res.Log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Database write failure.")
		}
	}

	d.writeWG.Done()
	return
}
