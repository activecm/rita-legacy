package parser

import (
	"sync"
	"time"

	"github.com/ocmdev/rita/database"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2"
)

type (
	// Document holds one item to be written to a database
	Document struct {
		Doc  ParsedDoc // Thing to write
		DB   string    // DB to write to
		Coll string    // Collection to write to
	}

	// DocWriter writes documents to a database
	DocWriter struct {
		ssn       *mgo.Session           // Session to db instance
		prefix    string                 // Prefix to the database
		importWl  bool                   // Flag to import whitelist
		whitelist []string               // Pointer to our whitelist array
		wchan     chan Document          // Document channel
		log       *log.Logger            // Logging
		wg        *sync.WaitGroup        // Used to block until complete
		meta      *database.MetaDBHandle // Handle to metadata
		databases []string               // Track the db states, cached
		dblock    *sync.Mutex            // For the databases fields
		started   bool                   // Track if we've started the writer
	}
)

// New generates a new DocWriter
func NewDocWriter(res *database.Resources) *DocWriter {
	return &DocWriter{
		ssn:       res.DB.Session,
		log:       res.Log,
		prefix:    res.System.BroConfig.DBPrefix,
		importWl:  res.System.ImportWhitelist,
		whitelist: res.System.Whitelist,
		wchan:     make(chan Document, 5000),
		wg:        new(sync.WaitGroup),
		meta:      res.MetaDB,
		databases: res.MetaDB.GetDatabases(),
		started:   false,
		dblock:    new(sync.Mutex)}
}

// Start begins the DocWriter spinning on its input
func (d *DocWriter) Start(count int) {
	// Add a second layer of protection against untracked starts.
	if !d.started {
		for i := 0; i < count; i++ {
			d.started = true
			go d.writeLoop()
		}
	}
	return
}

// Write allows a user to add to the channel
func (d *DocWriter) Write(doc Document) {
	doc.DB = d.prefix + doc.DB
	seen := false
	d.dblock.Lock()
	for _, aval := range d.databases {
		if aval == doc.DB {
			seen = true
		}
	}

	if !seen {
		d.meta.AddNewDB(doc.DB)
		d.databases = append(d.databases, doc.DB)
	}
	d.dblock.Unlock()
	d.wchan <- doc
	return
}

// Flush writes the final documents to the db and exits docwriter
func (d *DocWriter) Flush() {
	d.log.Debug("closing write channel")
	close(d.wchan)
	d.log.Debug("waiting for writes to finish")
	d.wg.Wait()
	d.ssn.Close()
	d.log.Debug("writes completed, exiting write loop")
	return
}

// writeLoop loops over the input channel spawning threads to write
func (d *DocWriter) writeLoop() {
	var err error
	d.wg.Add(1)
	ssn := d.ssn.Copy()
	defer ssn.Close()

	for {
		d.log.WithFields(log.Fields{
			"type":             "wldebug",
			"write_chan_count": len(d.wchan),
		}).Debug("WriteLoop status")
		doc, ok := <-d.wchan
		if !ok {
			d.log.Info("WriteLoop got closed channel, exiting")
			break
		}

		towrite := doc.Doc

		if !(d.importWl && towrite.IsWhiteListed(d.whitelist)) {
			err = ssn.DB(doc.DB).C(doc.Coll).Insert(towrite)
		}

		if err != nil {
			d.log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Database write failure")

			d.expFalloff(&doc)
		}
	}
	d.wg.Done()
	return
}

// expFalloff is entered after dbwrite failure
func (d *DocWriter) expFalloff(doc *Document) {
	for i := 0; i < 5; i++ {
		time.Sleep(time.Duration(i*i) * time.Second)
		ssn := d.ssn.Copy()
		defer ssn.Close()
		towrite := doc.Doc
		err := ssn.DB(doc.DB).C(doc.Coll).Insert(towrite)
		if err == nil {
			d.log.Info("Write succeeded")
			return
		}
		d.log.WithFields(log.Fields{
			"error":   err.Error(),
			"falloff": i,
		}).Error("Database write failure")
	}
}
