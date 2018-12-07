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
		// {Key: []string{"uconn_id"}, Unique: true},
		{Key: []string{"score"}},
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

	// beacons must be limited to a minimum of 20 connections, but if the user
	// selected a higher threshold, it will be used.
	thresh := util.Max(20, res.Config.S.Beacon.DefaultConnectionThresh)

	// copy session
	session := res.DB.Session.Copy()

	// create find query
	// first two lines: limit results to connection counts of min 20 and max 150k
	// third line: analysis needs at least four delta times to analyze
	// (Q1, Q2, Q3, Q4). This verifies at least 5 unique timestamps (connections may
	// result in duplicates, ts_list is a unique set)
	uconnsFindQuery := bson.M{
		"$and": []bson.M{
			bson.M{"connection_count": bson.M{"$gt": thresh}},
			bson.M{"connection_count": bson.M{"$lt": 150000}},
			bson.M{"ts_list.4": bson.M{"$exists": true}},
		}}

	// create result variable to receive query result
	var uconnRes struct {
		ID          bson.ObjectId `bson:"_id,omitempty"`
		Src         string        `bson:"src"`
		Dst         string        `bson:"dst"`
		TsList      []int64       `bson:"ts_list"`
		OrigIPBytes []int64       `bson:"orig_bytes_list"`
	}

	// execute query
	uconnIter := session.DB(res.DB.GetSelectedDB()).
		C(res.Config.T.Structure.UniqueConnTable).
		Find(uconnsFindQuery).Iter()

	// iterate over results and send to analysis worker
	for uconnIter.Next(&uconnRes) {
		// Note for Ethan: for some reason not doing the part below and reading directly into the
		// beacon analysis input structure with identically named fields and bson tags
		// causes incorrect information and huge errors, that's why its being typecast
		// below. Does not seem to impact performance
		newInput := &beacon.AnalysisInput{
			ID:          uconnRes.ID,
			Src:         uconnRes.Src,
			Dst:         uconnRes.Dst,
			TsList:      uconnRes.TsList,
			OrigIPBytes: uconnRes.OrigIPBytes,
		}
		analyzerWorker.analyze(newInput)
	}
	session.Close()

}

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
			{"$project", bson.D{
				{"ts_list", 1},
			}},
		},
		{
			{"$unwind", "$ts_list"},
		},
		{
			{"$sort", bson.D{
				{"ts_list", 1},
			}},
		},
		{
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
			{"$project", bson.D{
				{"ts_list", 1},
			}},
		},
		{
			{"$unwind", "$ts_list"},
		},
		{
			{"$sort", bson.D{
				{"ts_list", -1},
			}},
		},
		{
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

	// troubleshooting:
	//
	// fmt.Println("\t[-] First and last timestamps via uconns found: ")
	// fmt.Println("\t\t first: ", tsMin)
	// fmt.Println("\t\t last: ", tsMax)
	// fmt.Println("\t\t duration: ", time.Since(start))

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

// uconnsQuery := []bson.D{
// 	{
// 		{"$match", bson.D{
// 			{"connection_count", bson.D{
// 				{"$gt", thresh},
// 			}},
// 			{"connection_count", bson.D{
// 				{"$lt", 150000},
// 			}},
// 		}},
// 	},
// }
//
// uconnsFindQuery := bson.M{
// 	"$and": []bson.M{
// 		bson.M{"connection_count": bson.M{"$gt": thresh}},
// 		bson.M{"connection_count": bson.M{"$lt": 150000}},
// 	}}
//
// start1 := time.Now()
// count1, _ := res.DB.Session.DB(res.DB.GetSelectedDB()).
// 	C(res.Config.T.Structure.UniqueConnTable).
// 	Find(bson.M{"connection_count": bson.M{"$gt": thresh, "$lt": 150000}}).
// 	Count()
// dur1 := time.Since(start1)
// start2 := time.Now()
// count2, _ := res.DB.Session.DB(res.DB.GetSelectedDB()).
// 	C(res.Config.T.Structure.UniqueConnTable).
// 	Find(uconnsFindQuery).
// 	Count()
// dur2 := time.Since(start2)
// var temp []interface{}
//
// start3 := time.Now()
// _ = session.DB(res.DB.GetSelectedDB()).
// 	C(res.Config.T.Structure.UniqueConnTable).
// 	Pipe(uconnsQuery).All(&temp)
//
// count3 := len(temp)
//
// dur3 := time.Since(start3)
//
// fmt.Println("count1: ", count1, " duration: ", dur1)
// fmt.Println("count2: ", count2, " duration: ", dur2)
// fmt.Println("count3: ", count3, " duration: ", dur3)
