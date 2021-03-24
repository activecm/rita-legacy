package beaconproxy

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
)

type (
	dissector struct {
		connLimit         int64             // limit for strobe classification
		db                *database.DB      // provides access to MongoDB
		conf              *config.Config    // contains details needed to access MongoDB
		dissectedCallback func(*ProxyInput) // called on each analyzed result
		closedCallback    func()            // called when .close() is called and no more calls to analyzedCallback will be made
		dissectChannel    chan *ProxyInput  // holds unanalyzed data
		dissectWg         sync.WaitGroup    // wait for analysis to finish
	}
)

//newdissector creates a new collector for gathering data
func newDissector(connLimit int64, db *database.DB, conf *config.Config, dissectedCallback func(*ProxyInput), closedCallback func()) *dissector {
	return &dissector{
		connLimit:         connLimit,
		db:                db,
		conf:              conf,
		dissectedCallback: dissectedCallback,
		closedCallback:    closedCallback,
		dissectChannel:    make(chan *ProxyInput),
	}
}

//collect sends a chunk of data to be analyzed
func (d *dissector) collect(entry *ProxyInput) {
	d.dissectChannel <- entry
}

//close waits for the collector to finish
func (d *dissector) close() {
	close(d.dissectChannel)
	d.dissectWg.Wait()
	d.closedCallback()
}

//start kicks off a new analysis thread
func (d *dissector) start() {
	d.dissectWg.Add(1)
	go func() {
		ssn := d.db.Session.Copy()
		defer ssn.Close()

		for entry := range d.dissectChannel {

			// Check for errors and parse results
			// this is here because it will still return an empty document even if there are no results
			if entry.ConnectionCount > int64(d.conf.S.BeaconFQDN.DefaultConnectionThresh) {

				// check if strobe
				if entry.ConnectionCount > d.connLimit {
					// Set TsList to nil. The analyzer channel will check if this entry is nil and,
					// if so, will process it as a strobe
					entry.TsList = nil

					// set to writer channel
					d.dissectedCallback(entry)

				} else {
					// send to sorter channel if we have over UNIQUE 3 timestamps (analysis needs this verification)
					if len(entry.TsList) > 3 {
						d.dissectedCallback(entry)
					}
				}
			}
		}
		d.dissectWg.Done()
	}()
}
