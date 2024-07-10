package useragent

import (
	"strconv"
	"sync"

	"github.com/activecm/rita-legacy/config"
	"github.com/activecm/rita-legacy/database"
	"github.com/globalsign/mgo/bson"
)

// rareSignatureOrigIPsCutoff determines the cutoff for marking a particular IP as having used
// rare signature on an HTTP(s) connection. If a particular signature/ user agent is associated
// with less than `rareSignatureOrigIPsCutoff` originating IPs, we mark those IPs as having used
// a rare signature.
const rareSignatureOrigIPsCutoff = 5

type (
	//analyzer is a structure for useragent analysis
	analyzer struct {
		chunk            int                        //current chunk (0 if not on rolling analysis)
		chunkStr         string                     //current chunk (0 if not on rolling analysis)
		db               *database.DB               // provides access to MongoDB
		conf             *config.Config             // contains details needed to access MongoDB
		analyzedCallback func(database.BulkChanges) // called on each analyzed result
		closedCallback   func()                     // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Input                // holds unanalyzed data
		analysisWg       sync.WaitGroup             // wait for analysis to finish
	}
)

// newAnalyzer creates a new analyzer for recording connections that were made
// with HTTP useragents and TLS JA3 hashes
func newAnalyzer(chunk int, db *database.DB, conf *config.Config, analyzedCallback func(database.BulkChanges), closedCallback func()) *analyzer {
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

// collect gathers connection signature records for analysis
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
			useragentsSelector := bson.M{"user_agent": datum.Name}
			useragentsQuery := useragentsQuery(datum, a.chunk)
			a.analyzedCallback(database.BulkChanges{
				a.conf.T.UserAgent.UserAgentTable: []database.BulkChange{{
					Selector: useragentsSelector,
					Update:   useragentsQuery,
					Upsert:   true,
				}},
			})
		}

		a.analysisWg.Done()
	}()
}

// useragentsQuery returns a mgo query which inserts the given datum into the useragent collection. The useragent's
// originating IPs and requested FQDNs are capped in order to prevent hitting the MongoDB document size limits.
func useragentsQuery(datum *Input, chunk int) bson.M {
	origIPs := datum.OrigIps.Items()
	if len(origIPs) > 10 {
		origIPs = origIPs[:10]
	}

	requests := datum.Requests.Items()
	if len(requests) > 10 {
		requests = requests[:10]
	}

	return bson.M{
		"$push": bson.M{
			"dat": bson.M{
				"seen":     datum.Seen,
				"orig_ips": origIPs,
				"hosts":    requests,
				"cid":      chunk,
			},
		},
		"$set":         bson.M{"cid": chunk},
		"$setOnInsert": bson.M{"ja3": datum.JA3},
	}
}
