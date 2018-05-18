package beacon

import (
	"runtime"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/activecm/rita/database"
	"github.com/activecm/rita/datatypes/data"
	"github.com/activecm/rita/datatypes/structure"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"

	log "github.com/sirupsen/logrus"
)

type (
	//beaconAnalysisInput binds a src, dst pair with their analysis data
	beaconAnalysisInput struct {
		src         string        // Source IP
		dst         string        // Destination IP
		uconnID     bson.ObjectId // Unique Connection ID
		ts          []int64       // Connection timestamps for this src, dst pair
		origIPBytes []int64       // Src to dst connection sizes for each connection
	}
)

// BuildBeaconCollection pulls data from the unique connections collection
// and performs statistical analysis searching for command and control
// beacons.
func BuildBeaconCollection(res *resources.Resources) {
	// create the actual collection
	collectionName := res.Config.T.Beacon.BeaconTable
	collectionKeys := []mgo.Index{
		{Key: []string{"uconn_id"}, Unique: true},
		{Key: []string{"ts_score"}},
	}
	err := res.DB.CreateCollection(collectionName, false, collectionKeys)
	if err != nil {
		res.Log.Error("Failed: ", collectionName, err.Error())
		return
	}

	// If the threshold is incorrectly specified, fix it up.
	// We require at least four delta times to analyze
	// (Q1, Q2, Q3, Q4). So we need at least 5 connections
	thresh := res.Config.S.Beacon.DefaultConnectionThresh
	if thresh < 5 {
		thresh = 5
	}

	//Find the observation period
	minTime, maxTime := findAnalysisPeriod(
		res.DB,
		res.Config.T.Structure.ConnTable,
		res.Log,
	)

	//Create the workers
	writerWorker := newWriter(res.DB, res.Config)
	analyzerWorker := newAnalyzer(
		thresh, minTime, maxTime,
		writerWorker.write,
	)
	collectorWorker := newCollector(
		res.DB, res.Config, thresh,
		analyzerWorker.analyze,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		collectorWorker.start()
		analyzerWorker.start()
		writerWorker.start()
	}

	// Feed local addresses to the collector
	session := res.DB.Session.Copy()
	var host structure.Host
	localIter := session.DB(res.DB.GetSelectedDB()).
		C(res.Config.T.Structure.HostTable).
		Find(bson.M{"local": true}).Iter()

	for localIter.Next(&host) {
		collectorWorker.collect(host.IP)
	}
	session.Close()

	// Wait for things to finish
	res.Log.Debug("Finding all source / destination pairs for analysis")
	collectorWorker.flush()
	res.Log.Debug("Analyzing source / destination pairs")
	analyzerWorker.flush()
	res.Log.Debug("Finishing writing results to database")
	writerWorker.flush()
}

func findAnalysisPeriod(db *database.DB, connCollection string,
	logger *log.Logger) (int64, int64) {
	session := db.Session.Copy()
	defer session.Close()

	//Find first time
	logger.Debug("Looking for first connection timestamp")
	start := time.Now()

	//In practice having mongo do two lookups is
	//faster than pulling the whole collection
	//This could be optimized with an aggregation
	var conn data.Conn
	session.DB(db.GetSelectedDB()).
		C(connCollection).
		Find(nil).Limit(1).Sort("ts").Iter().Next(&conn)

	minTime := conn.Ts

	logger.Debug("Looking for last connection timestamp")
	session.DB(db.GetSelectedDB()).
		C(connCollection).
		Find(nil).Limit(1).Sort("-ts").Iter().Next(&conn)

	maxTime := conn.Ts

	logger.WithFields(log.Fields{
		"time_elapsed": time.Since(start),
		"first":        minTime,
		"last":         maxTime,
	}).Debug("First and last timestamps found")
	return minTime, maxTime
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
