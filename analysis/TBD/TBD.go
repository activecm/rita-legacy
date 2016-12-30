package TBD

import (
	"math"
	"sort"
	"sync"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/ocmdev/rita/config"
	datatype_TBD "github.com/ocmdev/rita/datatypes/TBD"
	"github.com/ocmdev/rita/datatypes/data"
	"github.com/ocmdev/rita/datatypes/structure"
	"github.com/ocmdev/rita/util"

	log "github.com/Sirupsen/logrus"
)

type (
	empty struct{}

	TBD struct {
		db                string // database
		resources         *config.Resources
		defaultConnThresh int // default connections threshold
		collectChannel    chan string
		analysisChannel   chan *tbdAnalysisInput
		writeChannel      chan *datatype_TBD.TBDAnalysisOutput
		collectWg         sync.WaitGroup
		analysisWg        sync.WaitGroup
		writeWg           sync.WaitGroup
		collectThreads    int
		analysisThreads   int
		writeThreads      int
		log               *log.Logger // system Logger
		minTime           int64       // minimum time
		maxTime           int64       // maximum time
	}

	tbdAnalysisInput struct {
		src     string
		dst     string
		uconnID bson.ObjectId
		ts      []int64
		//dur []int64
		//orig_bytes []int64
		//resp_bytes []int64
	}
)

// Name gives the name of this module
func New(c *config.Resources) *TBD {
	return &TBD{
		db:                c.System.DB,
		resources:         c,
		defaultConnThresh: c.System.TBDConfig.DefaultConnectionThresh,
		log:               c.Log,
		collectChannel:    make(chan string),
		analysisChannel:   make(chan *tbdAnalysisInput),
		writeChannel:      make(chan *datatype_TBD.TBDAnalysisOutput),
		collectThreads:    2,
		analysisThreads:   2,
		writeThreads:      2,
	}
}

//Start the TBD process
func (t *TBD) Run() {
	t.log.Info("Running beacon hunt")
	session := t.resources.Session.Copy()
	defer session.Close()

	//Find first time
	t.log.Debug("Looking for first connection timestamp")
	start := time.Now()

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

	// add local addresses to channel
	var host structure.Host
	localIter := session.DB(t.db).
		C(t.resources.System.StructureConfig.HostTable).
		Find(bson.M{"local": true}).Iter()

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

func (t *TBD) collect() {
	session := t.resources.Session.Copy()
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
			newInput := &tbdAnalysisInput{
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
			}
			t.analysisChannel <- newInput
		}
		host, more = <-t.collectChannel
	}
	t.collectWg.Done()
}

func (t *TBD) analyze() {
	data, more := <-t.analysisChannel
	for more {
		sort.Sort(util.SortableInt64(data.ts))

		//store the diff slice length since we use it a lot
		length := len(data.ts) - 1

		//find the duration of this connection
		//perfect beacons should fill the observation period
		duration := float64(data.ts[length]-data.ts[0]) /
			float64(t.maxTime-t.minTime)

		//find the delta times between the timestamps
		var diff []int64 = make([]int64, length)
		for i := 0; i < length; i++ {
			diff[i] = data.ts[i+1] - data.ts[i]
		}

		//perfect beacons should have symmetric delta time distributions
		//Bowley's measure of skew is used to check symmetry
		sort.Sort(util.SortableInt64(diff))
		bSkew := float64(0)
		low := diff[toInt64(.25*float64(length-1))]
		mid := diff[toInt64(.5*float64(length-1))]
		high := diff[toInt64(.75*float64(length-1))]
		bNum := low + high - 2*mid
		bDen := high - low
		if bDen != 0 {
			bSkew = float64(bNum) / float64(bDen)
		}

		//perfect beacons should have very low dispersion around the
		//median of their delta times
		//Median Absolute Deviation About the Median
		//is used to check dispersion
		devs := make([]int64, length)
		for i := 0; i < length; i++ {
			devs[i] = abs(diff[i] - mid)
		}

		sort.Sort(util.SortableInt64(devs))
		madm := devs[toInt64(.5*float64(length-1))]

		//Store the range for human analysis
		iRange := diff[length-1] - diff[0]

		//get a list of the intervals found in the data,
		//the number of times the interval was found,
		//and the most occurring interval
		intervals, intervalCounts, mode, modeCount := createCountMap(diff)

		newOutput := &datatype_TBD.TBDAnalysisOutput{
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
		//skew is mainly used as a tie breaker
		alpha := 1.0 - math.Abs(bSkew)

		//lower dispersion is better, cutoff dispersion scores at 30 seconds
		beta := 1.0 - float64(madm)/30.0
		if beta < 0 {
			beta = 0
		}
		gamma := duration

		//in order of ascending importance: skew, duration, dispersion
		newOutput.TS_score = (alpha + beta + gamma) / 3.0

		t.writeChannel <- newOutput
		data, more = <-t.analysisChannel
	}
	t.analysisWg.Done()
}

func (t *TBD) write() {
	session := t.resources.Session.Copy()
	defer session.Close()

	data, more := <-t.writeChannel
	for more {
		session.DB(t.db).C(t.resources.System.TBDConfig.TBDTable).Insert(data)
		data, more = <-t.writeChannel
	}
	t.writeWg.Done()
}

//returns a distinct data array, data count array, the mode,
//and the number of times the mode occured
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

func GetViewPipeline(r *config.Resources, cuttoff float64) []bson.D {
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
				{"src", "$uconn.src"},
				{"dst", "$uconn.dst"},
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

//TODO: Move these to util
func abs(a int64) int64 {
	if a >= 0 {
		return a
	}
	return -a
}

func toInt64(f float64) int64 {
	_, float := math.Modf(f)
	if float > .5 {
		return int64(math.Ceil(f))
	}
	return int64(f)
}
