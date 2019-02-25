package beacon

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/globalsign/mgo/bson"
)

type (
	dissector struct {
		connLimit         int64             // limit for strobe classification
		db                *database.DB      // provides access to MongoDB
		conf              *config.Config    // contains details needed to access MongoDB
		dissectedCallback func(*uconn.Pair) // called on each analyzed result
		closedCallback    func()            // called when .close() is called and no more calls to analyzedCallback will be made
		dissectChannel    chan *uconn.Pair  // holds unanalyzed data
		dissectWg         sync.WaitGroup    // wait for analysis to finish
	}
)

//newdissector creates a new collector for gathering data
func newDissector(connLimit int64, db *database.DB, conf *config.Config, dissectedCallback func(*uconn.Pair), closedCallback func()) *dissector {
	return &dissector{
		connLimit:         connLimit,
		db:                db,
		conf:              conf,
		dissectedCallback: dissectedCallback,
		closedCallback:    closedCallback,
		dissectChannel:    make(chan *uconn.Pair),
	}
}

//collect sends a chunk of data to be analyzed
func (d *dissector) collect(data *uconn.Pair) {
	d.dissectChannel <- data
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

		for data := range d.dissectChannel {
			ssn := d.db.Session.Copy()
			defer ssn.Close()

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
				bson.M{"$match": bson.M{"$and": []bson.M{
					bson.M{"src": data.Src},
					bson.M{"dst": data.Dst},
					bson.M{"strobe": bson.M{"$ne": true}},
				}}},
				bson.M{"$limit": 1},
				bson.M{"$project": bson.M{
					"src":    "$src",
					"dst":    "$dst",
					"ts":     "$dat.ts",
					"bytes":  "$dat.bytes",
					"count":  "$dat.count",
					"tbytes": "$dat.tbytes",
					"icerts": "$dat.icerts",
				}},
				bson.M{"$unwind": "$count"},
				bson.M{"$project": bson.M{
					"src":    1,
					"dst":    1,
					"ts":     1,
					"bytes":  1,
					"count":  bson.M{"$sum": "$count"},
					"tbytes": 1,
					"icerts": 1,
				}},
				bson.M{"$match": bson.M{"count": bson.M{"$gt": d.conf.S.Beacon.DefaultConnectionThresh}}},
				bson.M{"$unwind": "$tbytes"},
				bson.M{"$project": bson.M{
					"src":    1,
					"dst":    1,
					"ts":     1,
					"bytes":  1,
					"count":  1,
					"tbytes": bson.M{"$sum": "$tbytes"},
					"icerts": 1,
				}},
				bson.M{"$unwind": "$ts"},
				bson.M{"$unwind": "$ts"},
				bson.M{"$group": bson.M{
					"_id":    "$_id",
					"src":    bson.M{"$first": "$src"},
					"dst":    bson.M{"$first": "$dst"},
					"ts":     bson.M{"$addToSet": "$ts"},
					"bytes":  bson.M{"$first": "$bytes"},
					"count":  bson.M{"$first": "$count"},
					"tbytes": bson.M{"$first": "$tbytes"},
					"icerts": bson.M{"$first": "$icerts"},
				}},
				bson.M{"$unwind": "$bytes"},
				bson.M{"$unwind": "$bytes"},
				bson.M{"$group": bson.M{
					"_id":    "$_id",
					"src":    bson.M{"$first": "$src"},
					"dst":    bson.M{"$first": "$dst"},
					"ts":     bson.M{"$first": "$ts"},
					"bytes":  bson.M{"$push": "$bytes"},
					"count":  bson.M{"$first": "$count"},
					"tbytes": bson.M{"$first": "$tbytes"},
					"icerts": bson.M{"$first": "$icerts"},
				}},
				bson.M{"$unwind": "$icerts"},
				bson.M{"$unwind": "$icerts"},
				bson.M{"$group": bson.M{
					"_id":    "$_id",
					"src":    bson.M{"$first": "$src"},
					"dst":    bson.M{"$first": "$dst"},
					"ts":     bson.M{"$first": "$ts"},
					"bytes":  bson.M{"$first": "$bytes"},
					"count":  bson.M{"$first": "$count"},
					"tbytes": bson.M{"$first": "$tbytes"},
					"icerts": bson.M{"$push": "$icerts"},
				}},
				bson.M{"$project": bson.M{
					"_id":    "$_id",
					"src":    1,
					"dst":    1,
					"ts":     1,
					"bytes":  1,
					"count":  1,
					"tbytes": 1,
					"icerts": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$icerts", 0}}},
				}},
			}

			var res struct {
				Src    string  `bson:"src"`
				Dst    string  `bson:"dst"`
				Count  int64   `bson:"count"`
				Ts     []int64 `bson:"ts"`
				Bytes  []int64 `bson:"bytes"`
				TBytes int64   `bson:"tbytes"`
				ICerts int     `bson:"icerts"`
			}

			_ = ssn.DB(d.db.GetSelectedDB()).C(d.conf.T.Structure.UniqueConnTable).Pipe(uconnFindQuery).One(&res)

			// Check for errors and parse results
			// this is here because it will still return an empty document even if there are no results
			if res.Count > 0 {
				invalidCertFlag := false
				if res.ICerts > 0 {
					invalidCertFlag = true
				}
				analysisInput := &uconn.Pair{Src: res.Src, Dst: res.Dst, ConnectionCount: res.Count, TotalBytes: res.TBytes, InvalidCertFlag: invalidCertFlag}

				// check if uconn has become a strobe
				if analysisInput.ConnectionCount > d.connLimit {
					// set to writer channel
					d.dissectedCallback(analysisInput)

				} else { // otherwise, parse timestamps and orig ip bytes

					analysisInput.TsList = res.Ts
					analysisInput.OrigBytesList = res.Bytes

					// send to writer channel if we have over 3 timestamps (analysis needs this verification)
					if len(analysisInput.TsList) > 3 {
						d.dissectedCallback(analysisInput)
					}

				}

			}

		}
		d.dissectWg.Done()
	}()
}
