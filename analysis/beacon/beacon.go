package beacon

import (
	"math"
	"runtime"
	"sort"
	"sync"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/ocmdev/rita/database"
	dataBeacon "github.com/ocmdev/rita/datatypes/beacon"
	"github.com/ocmdev/rita/datatypes/data"
	"github.com/ocmdev/rita/datatypes/structure"
	"github.com/ocmdev/rita/util"

	log "github.com/sirupsen/logrus"
)

type (
	//Beacon contains methods for conducting a beacon hunt
	Beacon struct {
		db                string                                // current database
		resources         *database.Resources                   // holds the global config and DB layer
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
		orig_bytes []int64
		//resp_bytes []int64
	}
)

func BuildBeaconCollection(res *database.Resources) {
	collection_name := res.System.BeaconConfig.BeaconTable
	collection_keys := []string{"uconn_id", "ts_score"}
	err := res.DB.CreateCollection(collection_name, collection_keys)
	if err != nil {
		res.Log.Error("Failed: ", collection_name, err.Error())
		return
	}
	newBeacon(res).run()
}

func GetBeaconResultsView(res *database.Resources, ssn *mgo.Session, cutoffScore float64) *mgo.Iter {
	pipeline := getViewPipeline(res, cutoffScore)
	return res.DB.AggregateCollection(res.System.BeaconConfig.BeaconTable, ssn, pipeline)
}

// New creates a new beacon module
func newBeacon(res *database.Resources) *Beacon {

	// If the threshold is incorrectly specified, fix it up.
	// We require at least four delta times to analyze
	// (Q1, Q2, Q3, Q4). So we need at least 5 connections
	thresh := res.System.BeaconConfig.DefaultConnectionThresh
	if thresh < 5 {
		thresh = 5
	}

	return &Beacon{
		db:                res.DB.GetSelectedDB(),
		resources:         res,
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
	t.log.Info("Running beacon hunt")
	session := t.resources.DB.Session.Copy()
	defer session.Close()

	//Find first time
	t.log.Debug("Looking for first connection timestamp")
	start := time.Now()

	//In practice having mongo do two lookups is
	//faster than pulling the whole collection
	//This could be optimized with an aggregation
	var conn data.Conn
	session.DB(t.db).
		C(t.resources.System.StructureConfig.ConnTable).
		Find(nil).Limit(1).Sort("ts").Iter().Next(&conn)

	t.minTime = conn.Ts

	t.log.Debug("Looking for last connection timestamp")
	session.DB(t.db).
		C(t.resources.System.StructureConfig.ConnTable).
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
		C(t.resources.System.StructureConfig.HostTable).
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
		t.collectChannel <- host.Ip
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

// collect grabs all src, dst pairs and their connection data
func (t *Beacon) collect() {
	session := t.resources.DB.Session.Copy()
	defer session.Close()
	host, more := <-t.collectChannel
	for more {
		//grab all destinations related with this host
		var uconn structure.UniqueConnection
		destIter := session.DB(t.db).
			C(t.resources.System.StructureConfig.UniqueConnTable).
			Find(bson.M{"src": host}).Iter()

		for destIter.Next(&uconn) {
			//skip the connection pair if they are under the threshold
			if uconn.ConnectionCount < t.defaultConnThresh {
				continue
			}

			//create our new input
			newInput := &beaconAnalysisInput{
				uconnID: uconn.ID,
				src:     uconn.Src,
				dst:     uconn.Dst,
			}

			//Grab connection data
			var conn data.Conn
			connIter := session.DB(t.db).
				C(t.resources.System.StructureConfig.ConnTable).
				Find(bson.M{"id_origin_h": uconn.Src, "id_resp_h": uconn.Dst}).
				Iter()

			for connIter.Next(&conn) {
				newInput.ts = append(newInput.ts, conn.Ts)
				newInput.orig_bytes = append(newInput.orig_bytes, conn.OriginBytes)
			}
			t.analysisChannel <- newInput
		}
		host, more = <-t.collectChannel
	}
	t.collectWg.Done()
}

// analyze src, dst pairs with their connection data
func (t *Beacon) analyze() {
	for data := range t.analysisChannel {
		//sort the size and timestamps since they may have arrived out of order
		sort.Sort(util.SortableInt64(data.ts))
		sort.Sort(util.SortableInt64(data.orig_bytes))

		//remove subsecond communications
		//these will appear as beacons if we do not remove them
		//subsecond beacon finding *may* be implemented later on...
		data.ts = util.RemoveSortedDuplicates(data.ts)

		//If removing duplicates lowered the conn count under the threshold,
		//remove this data from the analysis
		if len(data.ts) < t.resources.System.BeaconConfig.DefaultConnectionThresh {
			continue
		}

		//store the diff slice length since we use it a lot
		//for timestamps this is one less then the data slice length
		//since we are calculating the times in between readings
		length := len(data.ts) - 1
		ds_length := len(data.orig_bytes)

		//find the duration of this connection
		//perfect beacons should fill the observation period
		duration := float64(data.ts[length]-data.ts[0]) /
			float64(t.maxTime-t.minTime)

		//find the delta times between the timestamps
		diff := make([]int64, length)
		for i := 0; i < length; i++ {
			diff[i] = data.ts[i+1] - data.ts[i]
		}

		//perfect beacons should have symmetric delta time and size distributions
		//Bowley's measure of skew is used to check symmetry
		sort.Sort(util.SortableInt64(diff))
		bSkew := float64(0)
		ds_bSkew := float64(0)

		//length -1 is used since diff is a zero based slice
		low := diff[util.Round(.25*float64(length-1))]
		mid := diff[util.Round(.5*float64(length-1))]
		high := diff[util.Round(.75*float64(length-1))]
		bNum := low + high - 2*mid
		bDen := high - low

		ds_low := data.orig_bytes[util.Round(.25*float64(ds_length-1))]
		ds_mid := data.orig_bytes[util.Round(.5*float64(ds_length-1))]
		ds_high := data.orig_bytes[util.Round(.75*float64(ds_length-1))]
		ds_bNum := ds_low + ds_high - 2*ds_mid
		ds_bDen := ds_high - ds_low

		//bSkew should equal zero if the denominator equals zero
		//bowley skew is unreliable if Q2 = Q1 or Q2 = Q3
		if bDen != 0 && mid != low && mid != high {
			bSkew = float64(bNum) / float64(bDen)
		}
		
		if ds_bDen != 0 {
			ds_bSkew = float64(ds_bNum) / float64(ds_bDen)
		}

		//perfect beacons should have very low dispersion around the
		//median of their delta times
		//Median Absolute Deviation About the Median
		//is used to check dispersion
		devs := make([]int64, length)
		for i := 0; i < length; i++ {
			devs[i] = util.Abs(diff[i] - mid)
		}
		
		ds_devs := make([]int64, ds_length)
		for i := 0; i < ds_length; i++ {
			ds_devs[i] = util.Abs(data.orig_bytes[i] - ds_mid)
		}

		sort.Sort(util.SortableInt64(devs))
		sort.Sort(util.SortableInt64(ds_devs))
		
		madm := devs[util.Round(.5*float64(length-1))]
		ds_madm := ds_devs[util.Round(.5*float64(ds_length-1))]

		//Store the range for human analysis
		iRange := diff[length-1] - diff[0]

		//get a list of the intervals found in the data,
		//the number of times the interval was found,
		//and the most occurring interval
		intervals, intervalCounts, mode, modeCount := createCountMap(diff)

		output := dataBeacon.BeaconAnalysisOutput{
			UconnID:           data.uconnID,
			TS_iSkew:          bSkew,
			TS_iDispersion:    madm,
			TS_duration:       duration,
			TS_iRange:         iRange,
			TS_iMode:          mode,
			TS_iModeCount:     modeCount,
			TS_intervals:      intervals,
			TS_intervalCounts: intervalCounts,
		}

		//more skewed distributions recieve a lower score
		//less skewed distributions recieve a higher score
		alpha := 1.0 - math.Abs(bSkew)
		ds_alpha := 1.0 - math.Abs(ds_bSkew)

		//lower dispersion is better, cutoff dispersion scores at 30 seconds
		beta := 1.0 - float64(madm)/30.0
		if beta < 0 {
			beta = 0
		}
		ds_beta := 1.0 - float64(ds_madm)
		if ds_beta < 0 {
			ds_beta = 0
		}		
		
		gamma := duration

		//in order of ascending importance: skew, duration, dispersion
		output.TS_score = (alpha + beta + gamma) / 3.0
		//in order of ascending importance: skew, dispersion
		output.DS_score = (ds_alpha + ds_beta) / 2.0
		t.writeChannel <- &output
	}

	t.analysisWg.Done()
}

// write writes the beacon analysis results to the database
func (t *Beacon) write() {
	session := t.resources.DB.Session.Copy()
	defer session.Close()

	for data := range t.writeChannel {
		session.DB(t.db).C(t.resources.System.BeaconConfig.BeaconTable).Insert(data)
	}
	t.writeWg.Done()
}

// createCountMap returns a distinct data array, data count array, the mode,
// and the number of times the mode occured
func createCountMap(data []int64) ([]int64, []int64, int64, int64) {
	//create interval counts for human analysis
	dataMap := make(map[int64]int64)
	for _, d := range data {
		dataMap[d]++
	}

	distinct := make([]int64, len(dataMap))
	counts := make([]int64, len(dataMap))

	i := 0
	for k, v := range dataMap {
		distinct[i] = k
		counts[i] = v
		i++
	}

	mode := distinct[0]
	max := counts[0]
	for idx, count := range counts {
		if count > max {
			max = count
			mode = distinct[idx]
		}
	}
	return distinct, counts, mode, max
}

// GetViewPipeline creates an aggregation for user views since the beacon table
// stores uconn uid's rather than src, dest pairs
func getViewPipeline(r *database.Resources, cuttoff float64) []bson.D {
	return []bson.D{
		{
			{"$match", bson.D{
				{"ts_score", bson.D{
					{"$gt", cuttoff},
				}},
			}},
		},
		{
			{"$lookup", bson.D{
				{"from", r.System.StructureConfig.UniqueConnTable},
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
				{"ts_score", -1},
			}},
		},
		{
			{"$project", bson.D{
				{"ts_score", 1},
				{"ds_score", 1},
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
			}},
		},
	}
}
