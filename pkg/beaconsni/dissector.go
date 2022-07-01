package beaconsni

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

type (
	//dissector gathers all of the connection details between a host and an SNI
	dissector struct {
		connLimit         int64                       // limit for strobe classification
		db                *database.DB                // provides access to MongoDB
		conf              *config.Config              // contains details needed to access MongoDB
		dissectedCallback func(dissectorResults)      // gathered SNI connection details are sent to this callback
		closedCallback    func()                      // called when .close() is called and no more calls to dissectedCallback will be made
		dissectChannel    chan data.UniqueSrcFQDNPair // holds data to be processed
		dissectWg         sync.WaitGroup              // wait for dissector to finish
	}
)

//newDissector creates a new dissector for gathering data
func newDissector(connLimit int64, db *database.DB, conf *config.Config, dissectedCallback func(dissectorResults), closedCallback func()) *dissector {
	return &dissector{
		connLimit:         connLimit,
		db:                db,
		conf:              conf,
		dissectedCallback: dissectedCallback,
		closedCallback:    closedCallback,
		dissectChannel:    make(chan data.UniqueSrcFQDNPair),
	}
}

//collect gathers a pair of hosts to obtain SNI connection data for
func (d *dissector) collect(datum data.UniqueSrcFQDNPair) {
	d.dissectChannel <- datum
}

//close waits for the dissector to finish
func (d *dissector) close() {
	close(d.dissectChannel)
	d.dissectWg.Wait()
	d.closedCallback()
}

//start kicks off a new dissector thread
func (d *dissector) start() {
	d.dissectWg.Add(1)
	go func() {
		ssn := d.db.Session.Copy()
		defer ssn.Close()

		for datum := range d.dissectChannel {

			matchNoStrobeKey := datum.BSONKey()

			// we are able to filter out already flagged strobes here
			// because we use the sniconns table to access them. The sniconns table has
			// already had its counts and stats updated.
			matchNoStrobeKey["dat.tls.strobe"] = bson.M{"$ne": true}
			matchNoStrobeKey["dat.http.strobe"] = bson.M{"$ne": true}

			// This will work for both updating and inserting completely new Beacons
			// for every new uconn record we have, we will check the uconns table. This
			// will always return a result because even with a brand new database, we already
			// created the uconns table. It will only continue and analyze if the connection
			// meets the required specs, again working for both an update and a new src-dst pair.
			// We would have to perform this check regardless if we want the rolling update
			// option to remain, and this gets us the vetting for both situations, and Only
			// works on the current entries - not a re-aggregation on the whole collection,
			// and individual lookups like this are really fast. This also ensures a unique
			// set of timestamps for analysis.
			uconnFindQuery := []bson.M{
				{"$match": matchNoStrobeKey},
				{"$limit": 1},
				{"$project": bson.M{
					"ts":     bson.M{"$concatArrays": []string{"$dat.http.ts", "$dat.tls.ts"}},
					"bytes":  bson.M{"$concatArrays": []string{"$dat.http.bytes", "$dat.tls.bytes"}},
					"count":  bson.M{"$concatArrays": []string{"$dat.http.count", "$dat.tls.count"}},
					"tbytes": bson.M{"$concatArrays": []string{"$dat.http.tbytes", "$dat.tls.tbytes"}},
				}},
				{"$unwind": "$count"},
				{"$group": bson.M{
					"_id":    "$_id",
					"ts":     bson.M{"$first": "$ts"},
					"bytes":  bson.M{"$first": "$bytes"},
					"count":  bson.M{"$sum": "$count"},
					"tbytes": bson.M{"$first": "$tbytes"},
				}},
				{"$match": bson.M{"count": bson.M{"$gt": d.conf.S.BeaconSNI.DefaultConnectionThresh}}},
				{"$unwind": "$tbytes"},
				{"$group": bson.M{
					"_id":    "$_id",
					"ts":     bson.M{"$first": "$ts"},
					"bytes":  bson.M{"$first": "$bytes"},
					"count":  bson.M{"$first": "$count"},
					"tbytes": bson.M{"$sum": "$tbytes"},
				}},
				{"$unwind": "$ts"},
				{"$unwind": "$ts"},
				{"$group": bson.M{
					"_id":    "$_id",
					"ts":     bson.M{"$addToSet": "$ts"},
					"bytes":  bson.M{"$first": "$bytes"},
					"count":  bson.M{"$first": "$count"},
					"tbytes": bson.M{"$first": "$tbytes"},
				}},
				{"$unwind": "$bytes"},
				{"$unwind": "$bytes"},
				{"$group": bson.M{
					"_id":    "$_id",
					"ts":     bson.M{"$first": "$ts"},
					"bytes":  bson.M{"$push": "$bytes"},
					"count":  bson.M{"$first": "$count"},
					"tbytes": bson.M{"$first": "$tbytes"},
				}},
				{"$project": bson.M{
					"_id":    "$_id",
					"ts":     1,
					"bytes":  1,
					"count":  1,
					"tbytes": 1,
				}},
			}

			var res struct {
				Count  int64   `bson:"count"`
				Ts     []int64 `bson:"ts"`
				Bytes  []int64 `bson:"bytes"`
				TBytes int64   `bson:"tbytes"`
			}

			_ = ssn.DB(d.db.GetSelectedDB()).C(d.conf.T.Structure.SNIConnTable).Pipe(uconnFindQuery).AllowDiskUse().One(&res)

			// Check for errors and parse results
			// this is here because it will still return an empty document even if there are no results
			if res.Count > 0 {
				analysisInput := dissectorResults{
					Hosts:           datum,
					ConnectionCount: res.Count,
					TotalBytes:      res.TBytes,
				}

				// check if uconn has become a strobe
				if analysisInput.ConnectionCount > d.connLimit {
					d.dissectedCallback(analysisInput)
				} else { // otherwise, parse timestamps and orig ip bytes
					analysisInput.TsList = res.Ts
					analysisInput.OrigBytesList = res.Bytes
					// the analysis worker requires that we have over UNIQUE 3 timestamps
					// we drop the input here since it is the earliest place in the pipeline to do so
					if len(analysisInput.TsList) > 3 {
						d.dissectedCallback(analysisInput)
					}
				}
			}
		}
		d.dissectWg.Done()
	}()
}
