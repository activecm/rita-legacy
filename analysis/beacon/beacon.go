package beacon

import (
	"runtime"
	"time"

	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"github.com/activecm/rita/database"
	"github.com/activecm/rita/datatypes/beacon"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"

	log "github.com/sirupsen/logrus"
)

// BuildBeaconCollection pulls data from the unique connections collection
// and performs statistical analysis searching for command and control
// beacons.
func BuildBeaconCollection(res *resources.Resources) {
	// create the actual collection
	collectionName := res.Config.T.Beacon.BeaconTable
	collectionKeys := []mgo.Index{
		{Key: []string{"score"}},
		{Key: []string{"$hashed:src"}},
		{Key: []string{"$hashed:dst"}},
		{Key: []string{"connection_count"}},
	}
	err := res.DB.CreateCollection(collectionName, collectionKeys)
	if err != nil {
		res.Log.Error("Failed: ", collectionName, err.Error())
		return
	}

	//Find the observation period
	minTime, maxTime := findAnalysisPeriod(
		res.DB,
		res.Config.T.Structure.UniqueConnTable,
		res.Log,
	)

	//Create the workers
	writerWorker := newWriter(res.DB, res.Config)
	analyzerWorker := newAnalyzer(
		minTime, maxTime,
		writerWorker.write, writerWorker.close,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		analyzerWorker.start()
		writerWorker.start()
	}

	// copy session
	session := res.DB.Session.Copy()

	// create find query
	// first two lines: limit results to connection counts of min 20 and max 150k
	// third line: analysis needs at least four delta times to analyze
	// (Q1, Q2, Q3, Q4). This verifies at least 5 unique timestamps (connections may
	// result in duplicates, ts_list is a unique set)
	uconnsFindQuery := bson.M{
		"$and": []bson.M{
			bson.M{"connection_count": bson.M{"$gt": res.Config.S.Beacon.DefaultConnectionThresh}},
			bson.M{"connection_count": bson.M{"$lt": 150000}},
			bson.M{"ts_list.4": bson.M{"$exists": true}},
		}}

	// create result variable to receive query result
	var uconnRes struct {
		ID              bson.ObjectId `bson:"_id,omitempty"`
		Src             string        `bson:"src"`
		Dst             string        `bson:"dst"`
		TsList          []int64       `bson:"ts_list"`
		OrigIPBytes     []int64       `bson:"orig_bytes_list"`
		ConnectionCount int           `bson:"connection_count"`
	}

	// execute query
	uconnIter := session.DB(res.DB.GetSelectedDB()).
		C(res.Config.T.Structure.UniqueConnTable).
		Find(uconnsFindQuery).Iter()

	// iterate over results and send to analysis worker
	for uconnIter.Next(&uconnRes) {
		// for some reason not doing the part below and reading directly into the
		// beacon analysis input structure with identically named fields and bson tags
		// causes incorrect information and huge errors, that's why it's being typecast
		// below. Does not seem to impact performance
		newInput := &beacon.AnalysisInput{
			ID:              uconnRes.ID,
			Src:             uconnRes.Src,
			Dst:             uconnRes.Dst,
			TsList:          uconnRes.TsList,
			OrigIPBytes:     uconnRes.OrigIPBytes,
			ConnectionCount: uconnRes.ConnectionCount,
		}
		analyzerWorker.analyze(newInput)
	}
	session.Close()

}

// findAnalysisPeriod returns the lowest and highest timestamps in the
// uconnCollection. These values are only used in calculating the
// duration metric. This implementation uses the uconn collection rather
// than conn because there could be many more entries in the conn collection
// which represent internal to internal or external to external traffic.
// These connections are filtered out before creating uconn. A more
// correct implementation might use conn, but duration's usefulness as a
// metric is currently under review and may be changing or going away shortly.
// A better solution, if these values are needed in the future, would be
// to calculate and store them in the database during import time or the
// creation of uconn collection.
func findAnalysisPeriod(db *database.DB, uconnCollection string,
	logger *log.Logger) (tsMin int64, tsMax int64) {
	session := db.Session.Copy()
	defer session.Close()

	// Structure to store the results of queries
	var res struct {
		Timestamp int64 `bson:"ts_list"`
	}

	// Get min timestamp
	logger.Debug("Looking for earliest (min) timestamp")
	start := time.Now()

	// Build query for aggregation
	tsMinQuery := []bson.D{
		{
			// include the "ts_list" field from every document
			{"$project", bson.D{
				{"ts_list", 1},
			}},
		},
		{
			// concat all the lists together
			{"$unwind", "$ts_list"},
		},
		{
			// sort all the timestamps from low to high
			{"$sort", bson.D{
				{"ts_list", 1},
			}},
		},
		{
			// take the lowest
			{"$limit", 1},
		},
	}

	// Execute query
	err := session.DB(db.GetSelectedDB()).C(uconnCollection).Pipe(tsMinQuery).One(&res)

	// Check for errors and parse results
	if err != nil {
		logger.Error("Error retrieving min ts info")
	} else {
		tsMin = res.Timestamp
	}

	// Get max timestamp
	logger.Debug("Looking for latest (max) timestamp")

	// Build query for aggregation
	tsMaxQuery := []bson.D{
		{
			// include the "ts_list" field from every document
			{"$project", bson.D{
				{"ts_list", 1},
			}},
		},
		{
			// concat all the lists together
			{"$unwind", "$ts_list"},
		},
		{
			// sort all the timestamps from high to low
			{"$sort", bson.D{
				{"ts_list", -1},
			}},
		},
		{
			// take the highest
			{"$limit", 1},
		},
	}

	// Execute query
	err2 := session.DB(db.GetSelectedDB()).C(uconnCollection).Pipe(tsMaxQuery).One(&res)

	// Check for errors and parse results
	if err2 != nil {
		logger.Error("Error retrieving min ts info")
	} else {
		tsMax = res.Timestamp
	}

	logger.WithFields(log.Fields{
		"time_elapsed": time.Since(start),
		"first":        tsMin,
		"last":         tsMax,
	}).Debug("First and last timestamps found")

	return
}

//GetBeaconResultsView finds beacons greater than a given cutoffScore
//and links the data from the unique connections table back in to the results
func GetBeaconResultsView(res *resources.Resources, ssn *mgo.Session, cutoffScore float64) *mgo.Iter {
	pipeline := getViewPipeline(res, cutoffScore)
	return res.DB.AggregateCollection(res.Config.T.Beacon.BeaconTable, ssn, pipeline)
}

// GetViewPipeline creates an aggregation for user views since the beacon collection
// stores uconn uid's rather than src, dest pairs. cuttoff is the lowest overall
// score to report on. Setting cuttoff to 0 retrieves all the records from the
// beaconing collection. Setting cuttoff to 1 will prevent the aggregation from
// returning any records.
func getViewPipeline(res *resources.Resources, cuttoff float64) []bson.D {
	return []bson.D{
		{
			{"$match", bson.D{
				{"score", bson.D{
					{"$gt", cuttoff},
				}},
			}},
		},
		{
			{"$lookup", bson.D{
				{"from", res.Config.T.Structure.UniqueConnTable},
				{"localField", "uconn_id"},
				{"foreignField", "_id"},
				{"as", "uconn"},
			}},
		},
		{
			{"$unwind", "$uconn"},
		},
		{
			{"$sort", bson.D{
				{"score", -1},
			}},
		},
		{
			{"$project", bson.D{
				{"score", 1},
				{"src", "$uconn.src"},
				{"dst", "$uconn.dst"},
				{"local_src", "$uconn.local_src"},
				{"local_dst", "$uconn.local_dst"},
				{"connection_count", "$uconn.connection_count"},
				{"avg_bytes", "$uconn.avg_bytes"},
				{"ts_iRange", 1},
				{"ts_iMode", 1},
				{"ts_iMode_count", 1},
				{"ts_iSkew", 1},
				{"ts_duration", 1},
				{"ts_iDispersion", 1},
				{"ds_dispersion", 1},
				{"ds_range", 1},
				{"ds_mode", 1},
				{"ds_mode_count", 1},
				{"ds_skew", 1},
			}},
		},
	}
}
