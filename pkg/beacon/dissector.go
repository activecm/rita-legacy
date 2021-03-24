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
		connLimit         int64              // limit for strobe classification
		db                *database.DB       // provides access to MongoDB
		conf              *config.Config     // contains details needed to access MongoDB
		dissectedCallback func(*uconn.Input) // called on each analyzed result
		closedCallback    func()             // called when .close() is called and no more calls to analyzedCallback will be made
		dissectChannel    chan *uconn.Input  // holds unanalyzed data
		dissectWg         sync.WaitGroup     // wait for analysis to finish
	}
)

//newdissector creates a new collector for gathering data
func newDissector(connLimit int64, db *database.DB, conf *config.Config, dissectedCallback func(*uconn.Input), closedCallback func()) *dissector {
	return &dissector{
		connLimit:         connLimit,
		db:                db,
		conf:              conf,
		dissectedCallback: dissectedCallback,
		closedCallback:    closedCallback,
		dissectChannel:    make(chan *uconn.Input),
	}
}

//collect sends a chunk of data to be analyzed
func (d *dissector) collect(datum *uconn.Input) {
	d.dissectChannel <- datum
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

		for datum := range d.dissectChannel {

			matchNoStrobeKey := datum.Hosts.BSONKey()

			// we are able to filter out already flagged strobes here
			// because we use the uconns table to access them. The uconns table has
			// already had its counts and stats updated. Note that this can't be the
			// case with the fqdn beacon strobes, as they are also stored in the fqdn
			// strobes collection and cannot be stored in uconns, needing the updated
			// counts and stats
			matchNoStrobeKey["strobe"] = bson.M{"$ne": true}

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
				bson.M{"$match": matchNoStrobeKey},
				bson.M{"$limit": 1},
				bson.M{"$project": bson.M{
					"ts":     "$dat.ts",
					"bytes":  "$dat.bytes",
					"count":  "$dat.count",
					"tbytes": "$dat.tbytes",
					"icerts": "$dat.icerts",
				}},
				bson.M{"$unwind": "$count"},
				bson.M{"$group": bson.M{
					"_id":    "$_id",
					"ts":     bson.M{"$first": "$ts"},
					"bytes":  bson.M{"$first": "$bytes"},
					"count":  bson.M{"$sum": "$count"},
					"tbytes": bson.M{"$first": "$tbytes"},
					"icerts": bson.M{"$first": "$icerts"},
				}},
				bson.M{"$match": bson.M{"count": bson.M{"$gt": d.conf.S.Beacon.DefaultConnectionThresh}}},
				bson.M{"$unwind": "$tbytes"},
				bson.M{"$group": bson.M{
					"_id":    "$_id",
					"ts":     bson.M{"$first": "$ts"},
					"bytes":  bson.M{"$first": "$bytes"},
					"count":  bson.M{"$first": "$count"},
					"tbytes": bson.M{"$sum": "$tbytes"},
					"icerts": bson.M{"$first": "$icerts"},
				}},
				bson.M{"$unwind": "$ts"},
				bson.M{"$unwind": "$ts"},
				bson.M{"$group": bson.M{
					"_id":    "$_id",
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
					"ts":     bson.M{"$first": "$ts"},
					"bytes":  bson.M{"$push": "$bytes"},
					"count":  bson.M{"$first": "$count"},
					"tbytes": bson.M{"$first": "$tbytes"},
					"icerts": bson.M{"$first": "$icerts"},
				}},
				bson.M{"$unwind": "$icerts"},
				bson.M{"$group": bson.M{
					"_id":    "$_id",
					"ts":     bson.M{"$first": "$ts"},
					"bytes":  bson.M{"$first": "$bytes"},
					"count":  bson.M{"$first": "$count"},
					"tbytes": bson.M{"$first": "$tbytes"},
					"icerts": bson.M{"$push": "$icerts"},
				}},
				bson.M{"$project": bson.M{
					"_id":    "$_id",
					"ts":     1,
					"bytes":  1,
					"count":  1,
					"tbytes": 1,
					"icerts": bson.M{"$anyElementTrue": []interface{}{"$icerts"}},
				}},
			}

			var res struct {
				Count  int64   `bson:"count"`
				Ts     []int64 `bson:"ts"`
				Bytes  []int64 `bson:"bytes"`
				TBytes int64   `bson:"tbytes"`
				ICerts bool    `bson:"icerts"`
			}

			_ = ssn.DB(d.db.GetSelectedDB()).C(d.conf.T.Structure.UniqueConnTable).Pipe(uconnFindQuery).AllowDiskUse().One(&res)

			// Check for errors and parse results
			// this is here because it will still return an empty document even if there are no results
			if res.Count > 0 {
				analysisInput := &uconn.Input{
					Hosts:           datum.Hosts,
					ConnectionCount: res.Count,
					TotalBytes:      res.TBytes,
					InvalidCertFlag: res.ICerts,
				}

				// check if uconn has become a strobe
				if analysisInput.ConnectionCount > d.connLimit {

					// set to sorter channel
					d.dissectedCallback(analysisInput)

				} else { // otherwise, parse timestamps and orig ip bytes

					analysisInput.TsList = res.Ts
					analysisInput.OrigBytesList = res.Bytes

					// send to sorter channel if we have over UNIQUE 3 timestamps (analysis needs this verification)
					if len(analysisInput.TsList) > 3 {
						d.dissectedCallback(analysisInput)
					}

				}
			}

		}
		d.dissectWg.Done()
	}()
}
