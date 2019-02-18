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
		db                *database.DB      // provides access to MongoDB
		conf              *config.Config    // contains details needed to access MongoDB
		dissectedCallback func(*uconn.Pair) // called on each analyzed result
		closedCallback    func()            // called when .close() is called and no more calls to analyzedCallback will be made
		dissectChannel    chan *uconn.Pair  // holds unanalyzed data
		dissectWg         sync.WaitGroup    // wait for analysis to finish
	}
)

//newdissector creates a new collector for gathering data
func newDissector(db *database.DB, conf *config.Config, dissectedCallback func(*uconn.Pair), closedCallback func()) *dissector {
	return &dissector{
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
					bson.M{"connection_count": bson.M{"$gt": d.conf.S.Beacon.DefaultConnectionThresh}},
					// bson.M{"connection_count": bson.M{"$lt": 150000}},
					bson.M{"strobe": bson.M{"$ne": true}},
				}}},
				bson.M{"$limit": 1},
			}
			var res uconnRes
			_ = ssn.DB(d.db.GetSelectedDB()).C(d.conf.T.Structure.UniqueConnTable).Pipe(uconnFindQuery).One(&res)

			// Check for errors and parse results
			// this is here because it will still return an empty document even if there are no results
			if len(res.Dat) > 0 {
				analysisInput := &uconn.Pair{Src: res.Src, Dst: res.Dst, ConnectionCount: res.ConnectionCount, TotalBytes: res.TotalBytes}

				// check if uconn has become a strobe
				if res.ConnectionCount > 2500 {
					// set to writer channel
					d.dissectedCallback(analysisInput)

				} else { // otherwise, parse timestamps and orig ip bytes
					for _, entry := range res.Dat {
						analysisInput.TsList = append(analysisInput.TsList, entry.Ts...)
						analysisInput.OrigBytesList = append(analysisInput.OrigBytesList, entry.OrigIPBytes...)
					}

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
