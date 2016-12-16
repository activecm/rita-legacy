package TBD

import (
	"math"
	"sort"
	"strconv"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/bglebrun/rita/config"
	datatype_TBD "github.com/bglebrun/rita/datatypes/TBD"
	"github.com/bglebrun/rita/datatypes/data"
	"github.com/bglebrun/rita/util"

	log "github.com/Sirupsen/logrus"
)

// Data coming from unique connection table aggregation
type (
	TBD struct {
		db        string // database
		resources *config.Resources
		log       *log.Logger // system logger

		batch_size          int     // BatchSize
		prefetch            float64 // Prefetch
		mintime             int64   // minimum time
		maxtime             int64   // maximum time
		default_bucket_size float64 // default size of buckets
		default_conn_thresh int     // default connections threshold
	}

	TBDInput struct {
		ID       bson.ObjectId `bson:"_id,omitempty"`
		Ts       []int64       `bson:"tss"`
		Src      string        `bson:"src"`
		Dst      string        `bson:"dst"`
		LocalSrc bool          `bson:"local_src"`
		LocalDst bool          `bson:"local_dst"`
		Dpts     []int         `bson:"dst_ports"`
		Dur      []float64     `bson:"duration"`
		Count    int           `bson:"connection_count"`
		Bytes    int64         `bson:"total_bytes"`
		BytesAvg float64       `bson:"avg_bytes"`
		Uid      []string      `bson:"uid"`
	}

	// hostType holds hosts for partial lookup
	hostType struct {
		ID    bson.ObjectId `bson:"_id"`
		IP    string        `bson:"ip"`
		Local bool          `bson:"local"`
	}

	// uniqueConn holds a unique connection element
	uniqueConn struct {
		ID            bson.ObjectId `bson:"_id"`
		Src           string        `bson:"src"`
		Dst           string        `bson:"dst"`
		ConnCount     int64         `bson:"connection_count"`
		LocalSrc      bool          `bson:"local_src"`
		LocalDst      bool          `bson:"local_dst"`
		TotalBytes    int64         `bson:"total_bytes"`
		AvgBytes      float64       `bson:"avg_bytes"`
		TotalDuration int64         `bson:"total_duration"`
	}

	// Key provides a structure for keying the lookups
	Key struct {
		ID  bson.ObjectId
		Src string
		Dst string
	}
)

// New produces a new TBD Module
func New(c *config.Resources) *TBD {
	return &TBD{
		db:                  c.System.DB,
		resources:           c,
		log:                 c.Log,
		batch_size:          c.System.BatchSize,
		prefetch:            c.System.Prefetch,
		default_bucket_size: c.System.TBDConfig.DefaultBucketSize,
		default_conn_thresh: c.System.TBDConfig.DefaultConnectionThresh,
	}
}

// Name gives the name of this module
func (t *TBD) Name() string { return "tbd" }

// Write data out to mongodb
func (t *TBD) writeData(
	key Key,
	rng int64,
	size int64,
	range_vals string,
	fill float64,
	spread float64,
	sum int64,
	score float64,
	intervalkeys []int64,
	intervalvals []int64,
	maxkey int64,
	maxval int64,
	tss []int64) {

	t.log.WithFields(log.Fields{
		"uconn_id": key.ID,
	}).Debug("Writing document")
	session := t.resources.Session.Copy()
	defer session.Close()

	// Create a write object
	obj := datatype_TBD.TBD{
		Src:           key.Src,
		Dst:           key.Dst,
		UconnID:       key.ID,
		Range:         rng,
		Size:          size,
		RangeVals:     range_vals,
		Fill:          fill,
		Spread:        spread,
		Sum:           sum,
		Score:         score,
		Intervals:     intervalkeys,
		InvervalCount: intervalvals,
		TopInterval:   maxkey,
		TopIntervalCt: maxval,
		Tss:           tss,
	}

	err := session.DB(t.db).C(t.resources.System.TBDConfig.TBDTable).Insert(obj)
	if err != nil {
		t.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Error writing TBD results to database")
	}
	t.log.Debug("Document written")
}

func (t *TBD) createIntervalMaps(timestampList []int64) (map[int64]int64, map[int64]int64, []int64) {

	// Get the end user selected bucket size
	bucket_size := t.default_bucket_size

	// Sort the timestamps in ascending order
	var ts_list util.SortableInt64
	ts_list = timestampList
	sort.Sort(ts_list)

	// Create a map that will act as a histogram of when connections occurred
	time_buckets := make(map[int64]int64)

	// Create a map that will act as a histogram for the amount of time between consecutive connections
	interval_buckets := make(map[int64]int64)

	// Create a list to hold sorted frequencies
	var interval_list []int64

	// Iterate over all the timestamps
	for i := 0; i < len(ts_list)-1; i++ {

		// Determine the number of seconds between two consecutive timestamps
		interval := ts_list[i+1] - ts_list[i]

		// Determine which time bucket the connection belongs in
		time := int64(float64(ts_list[i]-ts_list[0]) / bucket_size)

		// Counting the number of times timestamps show up in predetermined intervals of time
		_, timeFound := time_buckets[time]
		if timeFound {
			time_buckets[time]++
		} else {
			time_buckets[time]++
		}

		// Count the number of times specific frequencies show up
		if interval > 1 {
			_, freqFound := interval_buckets[interval]
			if freqFound {
				interval_buckets[interval]++
			} else {
				interval_buckets[interval]++
				interval_list = append(interval_list, interval)
			}
		}
	}

	return time_buckets, interval_buckets, interval_list
}

// TBDanalysis runs the TBD analysis on the dataset
func (t *TBD) TBDanalysis(timestampMap map[Key][]int64) {
	t.log.WithFields(log.Fields{
		"count": len(timestampMap),
	}).Debug("Entered analysis with count data points")

	// Get the end user selected bucket size
	bucket_size := t.default_bucket_size

	// Get the threshold for the number of connections that need to be seen before
	// a (src,dst,dpt) tuple is analyzed.
	connection_thresh := t.default_conn_thresh
	// Start result counter
	// written := 0

	var_map := make(map[Key]int64)
	sum_map := make(map[Key]int64)
	size_map := make(map[Key]int64)
	range_map := make(map[Key][]int64)
	fill_map := make(map[Key]int64)
	spread_map := make(map[Key]int64)

	// Iterate through the map
	for key, ts_list_temp := range timestampMap {

		// Sort the timestamps in ascending order
		var ts_list util.SortableInt64
		ts_list = ts_list_temp
		sort.Sort(ts_list)

		// Get the index for the last element in the list, since we use it in a few places
		last := len(ts_list) - 1
		first := 0

		// Find the length of time between the last and first timestamp for this connection
		sprd := ts_list[last] - ts_list[first]

		time_buckets, interval_buckets, interval_list := t.createIntervalMaps(ts_list)

		// Sort the keys (frequency intervals) for the frequency buckets in ascending order
		sort.Sort(util.SortableInt64(interval_list))

		// Number of frequencies. This will give us the number of frequency intervals
		interval_length := len(interval_list)

		// sum of values - This will give us the total number of connections
		interval_sum := int64(0)
		for i := 0; i < interval_length; i++ {
			interval_sum += interval_buckets[int64(interval_list[i])]
		}

		// If there were more than connection_thresh total connections...
		if len(ts_list) > connection_thresh && len(interval_list) > 0 {
			// best (smallest) range of key values that make up X% of the
			// freq_buckets histogram.
			best := int64(31536000) // (one year in seconds)

			best_sum := int64(0)

			// The keys in freq_buckets that make up the best range
			// for the variable above (best)
			best_range := []int64{0}

			// If there is only one frequency (ie every connection happens 2 seconds after the last)...
			// Return that one frequency
			if interval_length == 1 {
				best = 1
				best_range = []int64{interval_list[0]}
				best_sum = interval_buckets[int64(interval_list[0])]
			}

			// We are trying to find the smallest range of values that has the largest
			// "weight" in the histogram -- For example, trying to find the smallest
			// range that makes up 85% of the total data.

			// This loop iterates over the starting point for the range.
			for i := 0; i < interval_length-1; i++ {

				// Keep track of the sum of the number of connections having a freqeuncy
				// in the current range.
				mSum := interval_buckets[int64(interval_list[i])]

				// Keep track of the range between high and low frequencies
				mRange := int64(1)

				// Check if the current range contains X% of the total data
				if (float64(mSum)/float64(interval_sum) > 0.85) && (mRange < best) {
					best = mRange
					best_range = []int64{interval_list[i]}
					best_sum = mSum
				}

				// Increment the end of the range
				for j := i + 1; j < interval_length; j++ {
					mSum += interval_buckets[int64(interval_list[j])]
					mRange = interval_list[j] - interval_list[i]

					// Check if this range contains X% of the total data
					if (float64(mSum)/float64(interval_sum) > 0.85) && (mRange < best) {
						best = mRange
						best_range = []int64{interval_list[i], interval_list[j]}
						best_sum = mSum
					}
				}
			}
			var_map[key] = best
			size_map[key] = interval_sum //float64(len(ts_list))
			range_map[key] = best_range
			fill_map[key] = int64(len(time_buckets))
			spread_map[key] = sprd
			sum_map[key] = best_sum
		}
	}

	t.log.Debug("Entering the presence check..")
	for key, tsList := range timestampMap {
		present := true

		// Make sure all the necessary fields are available...
		if _, ok := var_map[key]; !ok {
			present = false
		}
		if _, ok := size_map[key]; !ok {
			present = false
		}
		if _, ok := range_map[key]; !ok {
			present = false
		}
		if _, ok := fill_map[key]; !ok {
			present = false
		}
		if _, ok := sum_map[key]; !ok {
			present = false
		}

		if present {
			t.log.Debug("Preparing to write data")
			// Extract all the necessary fields
			rng := var_map[key]
			size := size_map[key]
			range_vals := strconv.Itoa(int(range_map[key][0]))
			range_avg := float64(range_map[key][0])
			if len(range_map[key]) > 1 {
				range_min := strconv.Itoa(int(range_map[key][0]))
				range_max := strconv.Itoa(int(range_map[key][1]))
				range_vals = range_min + "--" + range_max
				range_avg = (range_avg + float64(range_map[key][1])) / 2.0
			}
			fill := float64(fill_map[key]) / (float64(t.maxtime-t.mintime) / bucket_size)
			spread := float64(spread_map[key]) / float64(t.maxtime-t.mintime)
			sum := sum_map[key]

			ideal_sum := float64(t.maxtime-t.mintime) / range_avg
			alpha := float64(sum) / ideal_sum
			if alpha > 1 {
				alpha = 1
			}
			alpha = 1 - alpha

			beta := float64(rng) / 600.0

			if beta > 1 {
				beta = 1
			}

			gamma := 1 - spread

			score := math.Sqrt(alpha*alpha + beta*beta + gamma*gamma)

			// Write out results with a score < 0
			if score < 1.0 {

				// Get interval lists for the interesting results
				_, interval_buckets, _ := t.createIntervalMaps(tsList)
				keys := make([]int64, 0, len(interval_buckets))
				vals := make([]int64, 0, len(interval_buckets))

				maxkey := int64(0)
				maxval := int64(0)
				for k, v := range interval_buckets {
					keys = append(keys, k)
					vals = append(vals, v)
					if v > maxval {
						maxval = v
						maxkey = k
					}
				}

				// Write the results.
				t.log.Debug("Writing to the database")
				t.writeData(key, rng, size, range_vals, fill*100, spread*100, sum, score, keys, vals, maxkey, maxval, tsList)
			}

		}

	}

}

// buildSet builds the data set to analyze
func (t *TBD) buildSet(hits *mgo.Iter, timestampMap map[Key][]int64) {
	t.log.Debug("Walking unique connections")
	var dat TBDInput
	// var dat map[string]interface{}

	for hits.Next(&dat) {
		thisKey := Key{
			ID:  dat.ID,
			Src: dat.Src,
			Dst: dat.Dst,
		}
		if len(dat.Ts) > t.default_conn_thresh {
			timestampMap[thisKey] = dat.Ts
		}
	}

}

// Run runst the module against current configuration
func (t *TBD) Run() {
	t.log.Info("starting beacon hunt")
	session := t.resources.Session.Copy()
	session.SetSocketTimeout(1 * time.Hour)
	defer session.Close()

	// Find First Time
	t.log.Debug("Looking for first time")
	start := time.Now()

	var d data.Conn
	var first int64 = 100000000000
	var last int64 = -1

	fliter := session.DB(t.db).C(t.resources.System.StructureConfig.ConnTable).
		Find(nil).
		Batch(t.batch_size).
		Prefetch(t.prefetch).
		Limit(1).
		Sort("ts").
		Iter()

	for fliter.Next(&d) {
		first = d.Ts
	}

	// Find Last Time
	t.log.Debug("Looking for last time")
	fliter2 := session.DB(t.db).C(t.resources.System.StructureConfig.ConnTable).
		Find(nil).
		Batch(t.batch_size).
		Prefetch(t.prefetch).
		Limit(1).
		Sort("-ts").
		Iter()

	for fliter2.Next(&d) {
		last = d.Ts
	}

	t.mintime = first
	t.maxtime = last
	t.log.WithFields(log.Fields{
		"time_elapsed": time.Since(start),
		"first":        t.mintime,
		"last":         t.maxtime,
	}).Debug("First and last found")

	// Create list of all local addresses
	var localAddresses []string
	var current hostType

	localIter := session.DB(t.db).C(t.resources.System.StructureConfig.HostTable).
		Find(bson.M{"local": true}).
		Iter()

	for localIter.Next(&current) {
		localAddresses = append(localAddresses, current.IP)
	}

	t.log.WithFields(log.Fields{
		"count": len(localAddresses),
	}).Info("found all local addresses")

	var keyList []Key

	for _, ipAddress := range localAddresses {
		var uc uniqueConn
		destIter := session.DB(t.db).
			C(t.resources.System.StructureConfig.UniqueConnTable).
			Find(bson.M{"src": ipAddress}).Iter()

		for destIter.Next(&uc) {
			if uc.ConnCount < int64(t.default_conn_thresh) {
				continue
			}

			newKey := Key{
				ID:  uc.ID,
				Src: uc.Src,
				Dst: uc.Dst,
			}

			keyList = append(keyList, newKey)
		}

	}

	t.log.WithFields(log.Fields{
		"count": len(keyList),
	}).Info("found all connections above threshold")

	tsMap := make(map[Key][]int64)

	longest := 0

	for _, currKey := range keyList {
		var curConn data.Conn
		tsIter := session.DB(t.db).
			C(t.resources.System.StructureConfig.ConnTable).
			Find(bson.M{"id_origin_h": currKey.Src, "id_resp_h": currKey.Dst}).
			Iter()

		for tsIter.Next(&curConn) {
			tsMap[currKey] = append(tsMap[currKey], curConn.Ts)
		}

		if longest < len(tsMap[currKey]) {
			longest = len(tsMap[currKey])
		}
	}

	t.log.WithFields(log.Fields{
		"most_connections": longest,
	}).Info("map is built, preparing analysis")
	t.TBDanalysis(tsMap)

	return
}
