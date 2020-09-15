package certificate

import (
	"strconv"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for exploded dns analysis
	analyzer struct {
		chunk            int            //current chunk (0 if not on rolling analysis)
		chunkStr         string         //current chunk (0 if not on rolling analysis)
		db               *database.DB   // provides access to MongoDB
		conf             *config.Config // contains details needed to access MongoDB
		analyzedCallback func(update)   // called on each analyzed result
		closedCallback   func()         // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Input    // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for parsing hostnames
func newAnalyzer(chunk int, db *database.DB, conf *config.Config, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		chunkStr:         strconv.Itoa(chunk),
		db:               db,
		conf:             conf,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *Input),
	}
}

//collect sends a group of domains to be analyzed
func (a *analyzer) collect(data *Input) {
	a.analysisChannel <- data
}

//close waits for the collector to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

//start kicks off a new analysis thread
func (a *analyzer) start() {
	a.analysisWg.Add(1)
	go func() {
		ssn := a.db.Session.Copy()
		defer ssn.Close()

		for data := range a.analysisChannel {
			// set up writer output
			var output update

			// cap the list to an arbitrary amount (hopefully smaller than the 16 MB document size cap)
			// anything approaching this limit will cause performance issues in software that depends on rita
			// anything tuncated over this limit won't be visible as an IP connecting to an invalid cert
			if len(data.OrigIps) > 200003 {
				data.OrigIps = data.OrigIps[:200003]
			}

			if len(data.Tuples) > 20 {
				data.Tuples = data.Tuples[:20]
			}

			if len(data.InvalidCerts) > 10 {
				data.InvalidCerts = data.InvalidCerts[:10]
			}

			//TODO[AGENT]: Use UniqueIP with NetworkID for orig_ips in certificate collection
			// create query
			query := bson.M{
				"$push": bson.M{
					"dat": bson.M{
						"seen":     data.Seen,
						"orig_ips": data.OrigIps,
						"tuples":   data.Tuples,
						"icodes":   data.InvalidCerts,
						"cid":      a.chunk,
					},
				},
				"$set": bson.M{"cid": a.chunk},
			}

			output.query = query

			output.collection = a.conf.T.Cert.CertificateTable

			//TODO[AGENT]: Use UniqueIP with NetworkID for host in certificate collection
			// create selector for output
			output.selector = bson.M{"host": data.Host}

			// set to writer channel
			a.analyzedCallback(output)

		}

		a.analysisWg.Done()
	}()
}
