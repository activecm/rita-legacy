package explodeddns

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	log "github.com/sirupsen/logrus"
)

type (
	//writer blah blah
	writer struct { //structure for writing blacklist results to mongo
		db           *database.DB   // provides access to MongoDB
		conf         *config.Config // contains details needed to access MongoDB
		log          *log.Logger    // main logger for RITA
		writeChannel chan update    // holds analyzed data
		writeWg      sync.WaitGroup // wait for writing to finish
	}
)

//newWriter creates a new writer object to write output data to blacklisted collections
func newWriter(db *database.DB, conf *config.Config, log *log.Logger) *writer {
	return &writer{
		db:           db,
		conf:         conf,
		log:          log,
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

			if data.newExplodedDNS.query != nil {
				info, err := ssn.DB(w.db.GetSelectedDB()).C(w.conf.T.DNS.ExplodedDNSTable).Upsert(
					data.newExplodedDNS.selector, data.newExplodedDNS.query,
				)

				if err != nil ||
					((info.Updated == 0) && (info.UpsertedId == nil)) {
					w.log.WithFields(log.Fields{
						"Module": "dns",
						"Info":   info,
						"Data":   data,
					}).Error(err)
				}
			}

			if data.existingExplodedDNS.query != nil {
				info, err := ssn.DB(w.db.GetSelectedDB()).C(w.conf.T.DNS.ExplodedDNSTable).UpdateWithArrayFilters(
					data.existingExplodedDNS.selector, data.existingExplodedDNS.query,
					data.existingExplodedDNS.arrayFilters, false,
				)

				if err != nil ||
					((info.Updated == 0) && (info.UpsertedId == nil)) {
					w.log.WithFields(log.Fields{
						"Module": "dns",
						"Info":   info,
						"Data":   data,
					}).Error(err)
				}
			}
		}
		w.writeWg.Done()
	}()
}
