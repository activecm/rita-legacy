package beacon

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type (
	//dissector gathers all of the unique connection details between pairs of hosts
	dissector struct {
		connLimit         int64             // limit for strobe classification
		chunk             int               // current chunk (0 if not on rolling analysis)
		db                *database.DB      // provides access to MongoDB
		conf              *config.Config    // contains details needed to access MongoDB
		dissectedCallback func(siphonInput) // gathered unique connection details are sent to this callback
		closedCallback    func()            // called when .close() is called and no more calls to dissectedCallback will be made
		dissectChannel    chan *uconn.Input // holds data to be processed
		dissectWg         sync.WaitGroup    // wait for dissector to finish
	}
)

// newDissector creates a new dissector for gathering data
func newDissector(connLimit int64, chunk int, db *database.DB, conf *config.Config, dissectedCallback func(siphonInput), closedCallback func()) *dissector {
	return &dissector{
		connLimit:         connLimit,
		chunk:             chunk,
		db:                db,
		conf:              conf,
		dissectedCallback: dissectedCallback,
		closedCallback:    closedCallback,
		dissectChannel:    make(chan *uconn.Input),
	}
}

// collect gathers a pair of hosts to obtain unique connection data for
func (d *dissector) collect(datum *uconn.Input) {
	d.dissectChannel <- datum
}

// close waits for the dissector to finish
func (d *dissector) close() {
	close(d.dissectChannel)
	d.dissectWg.Wait()
	d.closedCallback()
}

// start kicks off a new dissector thread
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
				{"$match": matchNoStrobeKey},
				{"$limit": 1},
				{"$project": bson.M{
					"ts":     "$dat.ts",
					"bytes":  "$dat.bytes",
					"count":  "$dat.count",
					"tbytes": "$dat.tbytes",
				}},
				{"$unwind": "$count"},
				{"$group": bson.M{
					"_id":    "$_id",
					"ts":     bson.M{"$first": "$ts"},
					"bytes":  bson.M{"$first": "$bytes"},
					"count":  bson.M{"$sum": "$count"},
					"tbytes": bson.M{"$first": "$tbytes"},
				}},
				{"$match": bson.M{"count": bson.M{"$gt": d.conf.S.Beacon.DefaultConnectionThresh}}},
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
					"_id":       "$_id",
					"ts_unique": bson.M{"$addToSet": "$ts"},
					"ts":        bson.M{"$push": "$ts"},
					"bytes":     bson.M{"$first": "$bytes"},
					"count":     bson.M{"$first": "$count"},
					"tbytes":    bson.M{"$first": "$tbytes"},
				}},
				{"$unwind": "$bytes"},
				{"$unwind": "$bytes"},
				{"$group": bson.M{
					"_id":       "$_id",
					"ts_unique": bson.M{"$first": "$ts_unique"},
					"ts":        bson.M{"$first": "$ts"},
					"bytes":     bson.M{"$push": "$bytes"},
					"count":     bson.M{"$first": "$count"},
					"tbytes":    bson.M{"$first": "$tbytes"},
				}},
				{"$project": bson.M{
					"_id":           "$_id",
					"ts_unique_len": bson.M{"$size": "$ts_unique"},
					"ts":            1,
					"bytes":         1,
					"count":         1,
					"tbytes":        1,
				}},
			}

			var res struct {
				Count       int64   `bson:"count"`
				TsUniqueLen int64   `bson:"ts_unique_len"`
				Ts          []int64 `bson:"ts"`
				Bytes       []int64 `bson:"bytes"`
				TBytes      int64   `bson:"tbytes"`
			}

			_ = ssn.DB(d.db.GetSelectedDB()).C(d.conf.T.Structure.UniqueConnTable).Pipe(uconnFindQuery).AllowDiskUse().One(&res)

			// Check for errors and parse results
			// this is here because it will still return an empty document even if there are no results
			if res.Count > 0 {

				// check if uconn has become a strobe
				if res.Count > d.connLimit {
					// if uconn became a strobe just from the current chunk, then we would not have received it here
					// as uconns upgrades itself to a strobe if its connection count met the strobe thresh for this chunk only

					// if uconn became a strobe during this chunk over its cummulative connection count over all chunks,
					// then we must upgrade it to a strobe and remove the timestamp and bytes arrays from the current chunk
					// or else the uconn document can grow to unacceptable sizes
					// these tasks are to be handled by the siphon prior to sorting & analysis
					var actions []evaporator
					// remove the bytes and ts arrays for the current chunk in the uconn document
					listRemover := evaporator{
						collection: d.conf.T.Structure.UniqueConnTable,
						selector:   datum.Hosts.BSONKey(),
						query: mgo.Change{
							Update: bson.M{"$unset": bson.M{"dat.$[elem].bytes": "", "dat.$[elem].ts": ""}},
						},
						arrayFilters: []bson.M{
							{"elem.cid": d.chunk},
						},
					}
					// set the uconn as a strobe
					// this must be done as uconns unsets its strobe flag if the current chunk doesnt meet
					// the strobe limit
					strobeUpdater := evaporator{
						collection: d.conf.T.Structure.UniqueConnTable,
						selector:   datum.Hosts.BSONKey(),
						query: mgo.Change{
							Update: bson.M{"$set": bson.M{"strobe": true}},
						},
					}
					// remove the uconn from the beacon table as its now a strobe
					beaconRemover := evaporator{
						collection: d.conf.T.Beacon.BeaconTable,
						selector:   datum.Hosts.BSONKey(),
						query: mgo.Change{
							Remove: true,
						},
					}
					actions = append(actions, listRemover, strobeUpdater, beaconRemover)

					siphonInput := siphonInput{
						Drain:     nil, // should be nil as we dont want to pass the strobe on to analysis
						Evaporate: actions,
					}
					d.dissectedCallback(siphonInput)

				} else { // otherwise, parse timestamps and orig ip bytes
					siphonInput := siphonInput{
						Drain: &uconn.Input{
							Hosts:              datum.Hosts,
							ConnectionCount:    res.Count,
							TotalBytes:         res.TBytes,
							TsList:             res.Ts,
							UniqueTsListLength: res.TsUniqueLen,
							OrigBytesList:      res.Bytes,
						},
						Evaporate: nil, // should be nil as we dont have anything to update or remove
					}

					// the analysis worker requires that we have over UNIQUE 3 timestamps
					// we drop the input here since it is the earliest place in the pipeline to do so
					if siphonInput.Drain.UniqueTsListLength > 3 {
						d.dissectedCallback(siphonInput)
					}
				}
			}
		}
		d.dissectWg.Done()
	}()
}
