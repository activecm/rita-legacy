package beaconfqdn

import (
	// "fmt"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/hostname"
	"github.com/globalsign/mgo/bson"
)

type (
	dissector struct {
		connLimit         int64                     // limit for strobe classification
		db                *database.DB              // provides access to MongoDB
		conf              *config.Config            // contains details needed to access MongoDB
		dissectedCallback func(*hostname.FqdnInput) // called on each analyzed result
		closedCallback    func()                    // called when .close() is called and no more calls to analyzedCallback will be made
		dissectChannel    chan *hostname.FqdnInput  // holds unanalyzed data
		dissectWg         sync.WaitGroup            // wait for analysis to finish
	}
)

//newdissector creates a new collector for gathering data
func newDissector(connLimit int64, db *database.DB, conf *config.Config, dissectedCallback func(*hostname.FqdnInput), closedCallback func()) *dissector {
	return &dissector{
		connLimit:         connLimit,
		db:                db,
		conf:              conf,
		dissectedCallback: dissectedCallback,
		closedCallback:    closedCallback,
		dissectChannel:    make(chan *hostname.FqdnInput),
	}
}

//collect sends a chunk of data to be analyzed
func (d *dissector) collect(datum *hostname.FqdnInput) {
	d.dissectChannel <- datum
}

//close waits for the collector to finish
func (d *dissector) close() {
	close(d.dissectChannel)
	d.dissectWg.Wait()
	// d.closedCallback()
}

//start kicks off a new analysis thread
func (d *dissector) start() {
	d.dissectWg.Add(1)
	go func() {
		ssn := d.db.Session.Copy()
		defer ssn.Close()

		for datum := range d.dissectChannel {

			// set up src match key (src ip + network uuid)
			srcMatchKey := datum.Src.SrcBSONKey()

			// fmt.Println("dissector src: ", srcMatchKey)

			var dstList []interface{}
			// create dst match query section
			for _, dst := range datum.ResolvedIPs {
				dstList = append(dstList, dst.DstBSONKey())
			}

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
				bson.M{"$match": srcMatchKey},
				bson.M{"$match": bson.M{"$or": dstList}},
				bson.M{"$project": bson.M{
					"src": 1,
					"ts": bson.M{
						"$reduce": bson.M{
							"input":        "$dat.ts",
							"initialValue": []interface{}{},
							"in":           bson.M{"$concatArrays": []interface{}{"$$value", "$$this"}},
						},
					},
					"bytes": bson.M{
						"$reduce": bson.M{
							"input":        "$dat.bytes",
							"initialValue": []interface{}{},
							"in":           bson.M{"$concatArrays": []interface{}{"$$value", "$$this"}},
						},
					},
					"count":  bson.M{"$sum": "$dat.count"},
					"tbytes": bson.M{"$sum": "$dat.tbytes"},
					"icerts": bson.M{"$anyElementTrue": []interface{}{"$dat.icerts"}},
				}},
				bson.M{"$group": bson.M{
					"_id":    "$src",
					"ts":     bson.M{"$push": "$ts"},
					"bytes":  bson.M{"$push": "$bytes"},
					"count":  bson.M{"$sum": "$count"},
					"tbytes": bson.M{"$sum": "$tbytes"},
					"icerts": bson.M{"$push": "$icerts"},
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
				bson.M{"$project": bson.M{
					"_id":    0,
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

				analysisInput := &hostname.FqdnInput{
					Host:            datum.Host,
					Src:             datum.Src,
					ConnectionCount: res.Count,
					TotalBytes:      res.TBytes,
					InvalidCertFlag: res.ICerts,
				}

				//  parse timestamps and orig ip bytes

				analysisInput.TsList = res.Ts
				analysisInput.OrigBytesList = res.Bytes

				// send to writer channel if we have over UNIQUE 3 timestamps (analysis needs this verification)
				if len(analysisInput.TsList) > 3 {
					d.dissectedCallback(analysisInput)
				}

			}

		}
		d.dissectWg.Done()
	}()
}
