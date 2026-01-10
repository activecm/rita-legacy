package certificate

import (
	"sync"

	"github.com/activecm/rita-legacy/config"
	"github.com/activecm/rita-legacy/database"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer is a structure for invalid certificate analysis
	analyzer struct {
		chunk            int                        // current chunk (0 if not on rolling analysis)
		db               *database.DB               // provides access to MongoDB
		conf             *config.Config             // contains details needed to access MongoDB
		analyzedCallback func(database.BulkChanges) // called on each analyzed result
		closedCallback   func()                     // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Input                // holds unanalyzed data
		analysisWg       sync.WaitGroup             // wait for analysis to finish
	}
)

// newAnalyzer creates a new analyzer for recording connections that were made
// with invalid certificates
func newAnalyzer(chunk int, db *database.DB, conf *config.Config, analyzedCallback func(database.BulkChanges), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		db:               db,
		conf:             conf,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *Input),
	}
}

// collect gathers invalid certificate connection records for analysis
func (a *analyzer) collect(datum *Input) {
	a.analysisChannel <- datum
}

// close waits for the analyzer to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

// start kicks off a new analysis thread
func (a *analyzer) start() {
	a.analysisWg.Add(1)
	go func() {
		ssn := a.db.Session.Copy()
		defer ssn.Close()

		for datum := range a.analysisChannel {
			// cap the list to an arbitrary amount (hopefully smaller than the 16 MB document size cap)
			// anything approaching this limit will cause performance issues in software that depends on rita
			// anything tuncated over this limit won't be visible as an IP connecting to an invalid cert
			origIPs := datum.OrigIps.Items()
			if len(origIPs) > 200003 {
				origIPs = origIPs[:200003]
			}

			tuples := datum.Tuples.Items()
			if len(tuples) > 20 {
				tuples = tuples[:20]
			}

			invalidCerts := datum.InvalidCerts.Items()
			if len(invalidCerts) > 10 {
				invalidCerts = invalidCerts[:10]
			}

			// create certificateQuery
			certificateQuery := bson.M{
				"$push": bson.M{
					"dat": bson.M{
						"seen":     datum.Seen,
						"orig_ips": origIPs,
						"tuples":   tuples,
						"icodes":   invalidCerts,
						"cid":      a.chunk,
					},
				},
				"$set": bson.M{
					"cid":          a.chunk,
					"network_name": datum.Host.NetworkName,
				},
			}

			// set to writer channel
			a.analyzedCallback(database.BulkChanges{
				a.conf.T.Cert.CertificateTable: []database.BulkChange{{
					Selector: datum.Host.BSONKey(),
					Update:   certificateQuery,
					Upsert:   true,
				}},
			})
		}

		a.analysisWg.Done()
	}()
}
