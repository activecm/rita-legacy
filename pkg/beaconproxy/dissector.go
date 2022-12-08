package beaconproxy

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/uconnproxy"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type (
	dissector struct {
		connLimit         int64                  // limit for strobe classification
		chunk             int                    // current chunk (0 if not on rolling analysis)
		db                *database.DB           // provides access to MongoDB
		conf              *config.Config         // contains details needed to access MongoDB
		dissectedCallback func(siphonInput)      // called on each analyzed result
		closedCallback    func()                 // called when .close() is called and no more calls to analyzedCallback will be made
		dissectChannel    chan *uconnproxy.Input // holds unanalyzed data
		dissectWg         sync.WaitGroup         // wait for analysis to finish
	}
)

// newdissector creates a new collector for gathering data
func newDissector(connLimit int64, chunk int, db *database.DB, conf *config.Config, dissectedCallback func(siphonInput), closedCallback func()) *dissector {
	return &dissector{
		connLimit:         connLimit,
		chunk:             chunk,
		db:                db,
		conf:              conf,
		dissectedCallback: dissectedCallback,
		closedCallback:    closedCallback,
		dissectChannel:    make(chan *uconnproxy.Input),
	}
}

// collect sends a chunk of data to be analyzed
func (d *dissector) collect(entry *uconnproxy.Input) {
	d.dissectChannel <- entry
}

// close waits for the collector to finish
func (d *dissector) close() {
	close(d.dissectChannel)
	d.dissectWg.Wait()
	d.closedCallback()
}

// start kicks off a new analysis thread
func (d *dissector) start() {
	d.dissectWg.Add(1)
	go func() {
		ssn := d.db.Session.Copy()
		defer ssn.Close()

		for datum := range d.dissectChannel {

			matchNoStrobeKey := datum.Hosts.BSONKey()

			// we are able to filter out already flagged strobes here
			// because we use the uconnproxy table to access them. The uconnproxy table has
			// already had its counts and stats updated.
			matchNoStrobeKey["strobeFQDN"] = bson.M{"$ne": true}

			// This will work for both updating and inserting completely new proxy beacons
			// for every new uconnproxy record we have, we will check the uconnproxy table. This
			// will always return a result because even with a brand new database, we already
			// created the uconnproxy table. It will only continue and analyze if the connection
			// meets the required specs, again working for both an update and a new src-fqdn pair.
			// We would have to perform this check regardless if we want the rolling update
			// option to remain, and this gets us the vetting for both situations, and Only
			// works on the current entries - not a re-aggregation on the whole collection,
			// and individual lookups like this are really fast. This also ensures a unique
			// set of timestamps for analysis.
			uconnProxyFindQuery := []bson.M{
				{"$match": matchNoStrobeKey},
				{"$limit": 1},
				{"$project": bson.M{
					"ts":    "$dat.ts",
					"count": "$dat.count",
				}},
				{"$unwind": "$count"},
				{"$group": bson.M{
					"_id":   "$_id",
					"ts":    bson.M{"$first": "$ts"},
					"count": bson.M{"$sum": "$count"},
				}},
				{"$match": bson.M{"count": bson.M{"$gt": d.conf.S.BeaconProxy.DefaultConnectionThresh}}},
				{"$unwind": "$ts"},
				{"$unwind": "$ts"},
				{"$group": bson.M{
					"_id":     "$_id",
					"ts":      bson.M{"$addToSet": "$ts"},
					"ts_full": bson.M{"$push": "$ts"},
					"count":   bson.M{"$first": "$count"},
				}},
				{"$project": bson.M{
					"_id":     "$_id",
					"ts":      1,
					"ts_full": 1,
					"count":   1,
				}},
			}

			var res struct {
				Count  int64   `bson:"count"`
				Ts     []int64 `bson:"ts"`
				TsFull []int64 `bson:"ts_full"`
			}

			_ = ssn.DB(d.db.GetSelectedDB()).C(d.conf.T.Structure.UniqueConnProxyTable).Pipe(uconnProxyFindQuery).AllowDiskUse().One(&res)

			// Check for errors and parse results
			// this is here because it will still return an empty document even if there are no results
			if res.Count > 0 {
				analysisInput := &uconnproxy.Input{
					Hosts:           datum.Hosts,
					Proxy:           datum.Proxy,
					ConnectionCount: res.Count,
				}

				// check if uconnproxy has become a strobe
				if analysisInput.ConnectionCount > d.connLimit {

					// if uconnproxy became a strobe just from the current chunk, then we would not have received it here
					// as uconnproxy upgrades itself to a strobe if its connection count met the strobe thresh for this chunk only

					// if uconnproxy became a strobe during this chunk over its cummulative connection count over all chunks,
					// then we must upgrade it to a strobe and remove the timestamp and bytes arrays from the current chunk
					// or else the uconnproxy document can grow to unacceptable sizes
					// these tasks are to be handled by the siphon prior to sorting & analysis
					var actions []evaporator
					// remove the ts array for the current chunk in the uconnproxy document
					listRemover := evaporator{
						collection: d.conf.T.Structure.UniqueConnProxyTable,
						selector:   datum.Hosts.BSONKey(),
						query: mgo.Change{
							Update: bson.M{"$unset": bson.M{"dat.$[elem].ts": ""}},
						},
						arrayFilters: []bson.M{
							{"elem.cid": d.chunk},
						},
					}
					// set the uconnproxy as a strobe
					// this must be done as uconnproxy unsets its strobe flag if the current chunk doesnt meet
					// the strobe limit
					strobeUpdater := evaporator{
						collection: d.conf.T.Structure.UniqueConnProxyTable,
						selector:   datum.Hosts.BSONKey(),
						query: mgo.Change{
							Update: bson.M{"$set": bson.M{"strobeFQDN": true}},
						},
					}
					// remove the uconnproxy from the beaconproxy table as its now a strobe
					beaconRemover := evaporator{
						collection: d.conf.T.BeaconProxy.BeaconProxyTable,
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

				} else { // otherwise, parse timestamps

					analysisInput.TsList = res.Ts
					analysisInput.TsListFull = res.TsFull

					// send to sorter channel if we have over UNIQUE 3 timestamps (analysis needs this verification)
					if len(analysisInput.TsList) > 3 {
						siphonInput := siphonInput{
							Drain:     analysisInput,
							Evaporate: nil, // should be nil as we dont have anything to update or remove
						}
						d.dissectedCallback(siphonInput)
					}

				}
			}
		}
		d.dissectWg.Done()
	}()
}
