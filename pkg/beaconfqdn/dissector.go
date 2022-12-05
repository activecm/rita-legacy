package beaconfqdn

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

type (
	//dissector gathers all of the unique connection details between a host and a fqdn
	dissector struct {
		connLimit         int64            // limit for strobe classification
		db                *database.DB     // provides access to MongoDB
		conf              *config.Config   // contains details needed to access MongoDB
		dissectedCallback func(*fqdnInput) // gathered unique connection details are sent to this callback
		closedCallback    func()           // called when .close() is called and no more calls to analyzedCallback will be made
		dissectChannel    chan *fqdnInput  // holds data to be processed
		dissectWg         sync.WaitGroup   // wait for analysis to finish
	}
)

// newdissector creates a new dissector for gathering data
func newDissector(connLimit int64, db *database.DB, conf *config.Config, dissectedCallback func(*fqdnInput), closedCallback func()) *dissector {
	return &dissector{
		connLimit:         connLimit,
		db:                db,
		conf:              conf,
		dissectedCallback: dissectedCallback,
		closedCallback:    closedCallback,
		dissectChannel:    make(chan *fqdnInput),
	}
}

// collect gathers a resolved FQDN to obtain unique connection data for
func (d *dissector) collect(entry *fqdnInput) {
	d.dissectChannel <- entry
}

// close waits for the dissector to finish
func (d *dissector) close() {
	close(d.dissectChannel)
	d.dissectWg.Wait()
	d.closedCallback()
}

/*
db.getCollection('uconn').aggregate([
    {"$match": {
        "$or": [{"dst":"172.217.4.226"}]
    }},
    {"$project": {
        "src":              1,
        "src_network_uuid": 1,
        "src_network_name": 1,
        "ts": {
            "$reduce": {
                "input":            "$dat.ts",
                    "initialValue": [],
                    "in":           {"$concatArrays": ["$$value", "$$this"]},
            },
        },
        "bytes": {
            "$reduce": {
                "input":        "$dat.bytes",
                "initialValue": [],
                "in":           {"$concatArrays": ["$$value", "$$this"]},
            },
        },
        "count":  {"$sum": "$dat.count"},
        "tbytes": {"$sum": "$dat.tbytes"},
    }},
    {"$group": {
        "_id":              {"src": "$src", "uuid": "$src_network_uuid"},
        "ts":               {"$push": "$ts"},
        "bytes":            {"$push": "$bytes"},
        "count":            {"$sum": "$count"},
        "tbytes":           {"$sum": "$tbytes"},
        "src_network_name": {"$last": "$src_network_name"},
    }},
    {"$match": {"count": {"$gt": 20}}},
    {"$unwind": {
        "path": "$ts",
        // by default, $unwind does not output a document if the field value is null,
        // missing, or an empty array. Since uconns stops storing ts and byte array
        // results if a result is going to be guaranteed to be a beacon, we need this
        // to not discard the result so we can update the fqdn beacon accurately
        "preserveNullAndEmptyArrays": true,
    }},
    {"$unwind": {
        "path":                       "$ts",
        "preserveNullAndEmptyArrays": true,
    }},
    {"$group": {
        "_id": "$id",
        // need to unique-ify timestamps or else results
        // will be skewed by "0 distant" data points
        "ts":      {"$addToSet": "$ts"},
		"ts_full": {"$push": "$ts"},
        "bytes":   {"$first": "$bytes"},
        "count":   {"$first": "$count"},
        "tbytes":  {"$first": "$tbytes"},
        "src_network_name": {"$last": "$src_network_name"},
    }},
    {"$unwind": {
        "path":                       "$bytes",
        "preserveNullAndEmptyArrays": true,
    }},
    {"$unwind": {
        "path":                       "$bytes",
        "preserveNullAndEmptyArrays": true,
    }},
    {"$group": {
        "_id":     "$_id",
        "ts":      {"$first": "$ts"},
		"ts_full": {"$first": "$ts_full"},
        "bytes":   {"$push": "$bytes"},
        "count":   {"$first": "$count"},
        "tbytes":  {"$first": "$tbytes"},
        "src_network_name": {"$last": "$src_network_name"},
    }},
    {"$project": {
        "_id":              0,
        "src":              "$_id.src",
        "src_network_uuid": "$_id.uuid",
        "src_network_name": 1,
        "ts":               1,
		"ts_full":          1,
        "bytes":            1,
        "count":            1,
        "tbytes":           1,
    }},
])
*/
//start kicks off a new dissector thread
func (d *dissector) start() {
	d.dissectWg.Add(1)

	go func() {
		ssn := d.db.Session.Copy()
		defer ssn.Close()

		for entry := range d.dissectChannel {
			// This will work for both updating and inserting completely new Beacons
			// for every new hostnames record we have, we will check every entry in the
			// uconn table where the source IP from the hostnames record connected to one
			// of the associated IPs for  FQDN. This
			// will always return a result because even with a brand new database, we already
			// created the uconns table. It will only continue and analyze if the connection
			// meets the required specs, again working for both an update and a new src-fqdn
			// pair. We would have to perform this check regardless if we want the rolling
			// update option to remain, and this gets us the vetting for both situations, and
			// Only works on the current entries - not a re-aggregation on the whole collection,
			// and individual lookups like this are really fast. This also ensures a unique
			// set of timestamps for analysis.
			uconnFindQuery := []bson.M{
				// beacons strobe ignores any already flagged strobes, but we don't want to do
				// that here. Beacons relies on the uconn table for having the updated connection info
				// we do not have that, so the calculation must happen. We don't necessarily need to store
				// the tslist or byte list, but I don't think that leaving it in will significantly impact
				// performance on a few strobes.

				// This query pulls out all uconn entries where any of the resolved IPs in DstBSONList
				// are shown as a destination. We then group on unique Src (src IP, uuid, network name).
				// This returns an array of results such that for each Src, we have the timestamps from
				// that Src to any of the resolved IPs. We then iterate over that array of results to
				// perfrom the beacon FQDN analysis. This was shown to be over 8x faster than making
				// separate queries to the uconn table for each Src.
				{"$match": bson.M{"$or": entry.DstBSONList}},
				{"$project": bson.M{
					"src":              1,
					"src_network_uuid": 1,
					"src_network_name": 1,
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
				}},
				{"$group": bson.M{
					"_id":              bson.M{"src": "$src", "uuid": "$src_network_uuid"},
					"ts":               bson.M{"$push": "$ts"},
					"bytes":            bson.M{"$push": "$bytes"},
					"count":            bson.M{"$sum": "$count"},
					"tbytes":           bson.M{"$sum": "$tbytes"},
					"src_network_name": bson.M{"$last": "$src_network_name"},
				}},
				{"$match": bson.M{"count": bson.M{"$gt": d.conf.S.BeaconFQDN.DefaultConnectionThresh}}},
				{"$unwind": bson.M{
					"path": "$ts",
					// by default, $unwind does not output a document if the field value is null,
					// missing, or an empty array. Since uconns stops storing ts and byte array
					// results if a result is going to be guaranteed to be a beacon, we need this
					// to not discard the result so we can update the fqdn beacon accurately
					"preserveNullAndEmptyArrays": true,
				}},
				{"$unwind": bson.M{
					"path":                       "$ts",
					"preserveNullAndEmptyArrays": true,
				}},
				{"$group": bson.M{
					"_id": "$_id",
					// need to unique-ify timestamps or else results
					// will be skewed by "0 distant" data points (for bowley skew)
					"ts":               bson.M{"$addToSet": "$ts"},
					"ts_full":          bson.M{"$push": "$ts"},
					"bytes":            bson.M{"$first": "$bytes"},
					"count":            bson.M{"$first": "$count"},
					"tbytes":           bson.M{"$first": "$tbytes"},
					"src_network_name": bson.M{"$last": "$src_network_name"},
				}},
				{"$unwind": bson.M{
					"path":                       "$bytes",
					"preserveNullAndEmptyArrays": true,
				}},
				{"$unwind": bson.M{
					"path":                       "$bytes",
					"preserveNullAndEmptyArrays": true,
				}},
				{"$group": bson.M{
					"_id":              "$_id",
					"ts":               bson.M{"$first": "$ts"},
					"ts_full":          bson.M{"$first": "$ts_full"},
					"bytes":            bson.M{"$push": "$bytes"},
					"count":            bson.M{"$first": "$count"},
					"tbytes":           bson.M{"$first": "$tbytes"},
					"src_network_name": bson.M{"$last": "$src_network_name"},
				}},
				{"$project": bson.M{
					"_id":              0,
					"src":              "$_id.src",
					"src_network_uuid": "$_id.uuid",
					"src_network_name": 1,
					"ts":               1,
					"ts_full":          1,
					"bytes":            1,
					"count":            1,
					"tbytes":           1,
				}},
			}

			type (
				indvidualRes struct {
					Src            string      `bson:"src"`
					SrcNetworkUUID bson.Binary `bson:"src_network_uuid"`
					SrcNetworkName string      `bson:"src_network_name"`
					Count          int64       `bson:"count"`
					Ts             []int64     `bson:"ts"`
					TsFull         []int64     `bson:"ts_full"`
					Bytes          []int64     `bson:"bytes"`
					TBytes         int64       `bson:"tbytes"`
				}
			)

			var allResults []indvidualRes

			_ = ssn.DB(d.db.GetSelectedDB()).C(d.conf.T.Structure.UniqueConnTable).Pipe(uconnFindQuery).AllowDiskUse().All(&allResults)

			// Iterate through the results to run the analysis on each set of timestamps
			// between a Src and any of the resolved IPs for the current FQDN
			for _, res := range allResults {

				srcCurr := data.UniqueSrcIP{SrcIP: res.Src, SrcNetworkUUID: res.SrcNetworkUUID, SrcNetworkName: res.SrcNetworkName}
				analysisInput := &fqdnInput{
					FQDN:            entry.FQDN,
					Src:             srcCurr,
					ConnectionCount: res.Count,
					TotalBytes:      res.TBytes,
					ResolvedIPs:     entry.ResolvedIPs,
				}

				// check if beacon has become a strobe
				if analysisInput.ConnectionCount > d.connLimit {
					// we do not need to siphon the uconn if it is a strobe
					// because the beacon module already did this
					d.dissectedCallback(analysisInput)

				} else { // otherwise, parse timestamps and orig ip bytes
					analysisInput.TsList = res.Ts
					analysisInput.TsListFull = res.TsFull
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
