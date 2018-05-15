package beacon

import (
	"runtime"
	"sync"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	dataBeacon "github.com/activecm/rita/datatypes/beacon"
	"github.com/activecm/rita/datatypes/data"
	"github.com/activecm/rita/datatypes/structure"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"

	log "github.com/sirupsen/logrus"
)

type (
	//Beacon contains methods for conducting a beacon hunt
	Beacon struct {
		db                string                                // current database
		res               *resources.Resources                  // holds the global config and DB layer
		defaultConnThresh int                                   // default connections threshold
		collectChannel    chan string                           // holds ip addresses
		analysisChannel   chan *beaconAnalysisInput             // holds unanalyzed data
		writeChannel      chan *dataBeacon.BeaconAnalysisOutput // holds analyzed data
		collectWg         sync.WaitGroup                        // wait for collection to finish
		analysisWg        sync.WaitGroup                        // wait for analysis to finish
		writeWg           sync.WaitGroup                        // wait for writing to finish
		collectThreads    int                                   // the number of read / collection threads
		analysisThreads   int                                   // the number of analysis threads
		writeThreads      int                                   // the number of write threads
		log               *log.Logger                           // system Logger
		minTime           int64                                 // minimum time
		maxTime           int64                                 // maximum time
	}

	//beaconAnalysisInput binds a src, dst pair with their analysis data
	beaconAnalysisInput struct {
		src     string        // Source IP
		dst     string        // Destination IP
		uconnID bson.ObjectId // Unique Connection ID
		ts      []int64       // Connection timestamps for this src, dst pair
		//dur []int64
		origIPBytes []int64
		//resp_bytes []int64
	}
)

// BuildBeaconCollection pulls data from the unique connections collection
// and performs statistical analysis searching for command and control
// beacons.
func BuildBeaconCollection(res *resources.Resources) {
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
	newBeacon(res).run()
}

// New creates a new beacon module
func newBeacon(res *resources.Resources) *Beacon {

	// If the threshold is incorrectly specified, fix it up.
	// We require at least four delta times to analyze
	// (Q1, Q2, Q3, Q4). So we need at least 5 connections
	thresh := res.Config.S.Beacon.DefaultConnectionThresh
	if thresh < 5 {
		thresh = 5
	}

	return &Beacon{
		db:                res.DB.GetSelectedDB(),
		res:               res,
		defaultConnThresh: thresh,
		log:               res.Log,
		collectChannel:    make(chan string),
		analysisChannel:   make(chan *beaconAnalysisInput),
		writeChannel:      make(chan *dataBeacon.BeaconAnalysisOutput),
		collectThreads:    util.Max(1, runtime.NumCPU()/2),
		analysisThreads:   util.Max(1, runtime.NumCPU()/2),
		writeThreads:      util.Max(1, runtime.NumCPU()/2),
	}
}

// Run Starts the beacon hunt process
func (t *Beacon) run() {
	session := t.res.DB.Session.Copy()
	defer session.Close()

	//Find first time
	t.log.Debug("Looking for first connection timestamp")
	start := time.Now()

	//In practice having mongo do two lookups is
	//faster than pulling the whole collection
	//This could be optimized with an aggregation
	var conn data.Conn
	session.DB(t.db).
		C(t.res.Config.T.Structure.ConnTable).
		Find(nil).Limit(1).Sort("ts").Iter().Next(&conn)

	t.minTime = conn.Ts

	t.log.Debug("Looking for last connection timestamp")
	session.DB(t.db).
		C(t.res.Config.T.Structure.ConnTable).
		Find(nil).Limit(1).Sort("-ts").Iter().Next(&conn)

	t.maxTime = conn.Ts

	t.log.WithFields(log.Fields{
		"time_elapsed": time.Since(start),
		"first":        t.minTime,
		"last":         t.maxTime,
	}).Debug("First and last timestamps found")

	// add local addresses to collect channel
	var host structure.Host
	localIter := session.DB(t.db).
		C(t.res.Config.T.Structure.HostTable).
		Find(bson.M{"local": true}).Iter()

	//kick off the threaded goroutines
	for i := 0; i < t.collectThreads; i++ {
		t.collectWg.Add(1)
		go t.collect()
	}

	for i := 0; i < t.analysisThreads; i++ {
		t.analysisWg.Add(1)
		go t.analyze()
	}

	for i := 0; i < t.writeThreads; i++ {
		t.writeWg.Add(1)
		go t.write()
	}

	for localIter.Next(&host) {
		t.collectChannel <- host.IP
	}
	t.log.Debug("Finding all source / destination pairs for analysis")
	close(t.collectChannel)
	t.collectWg.Wait()
	t.log.Debug("Analyzing source / destination pairs")
	close(t.analysisChannel)
	t.analysisWg.Wait()
	t.log.Debug("Finishing writing results to database")
	close(t.writeChannel)
	t.writeWg.Wait()
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
