package beacon

import (
	"math"
	"sort"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/util"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (
	//analyzer handles calculating statistical measures of the distributions of the
	//timestamps and data sizes between pairs of hosts
	analyzer struct {
		tsMin            int64                // min timestamp for the whole dataset
		tsMax            int64                // max timestamp for the whole dataset
		chunk            int                  // current chunk (0 if not on rolling analysis)
		db               *database.DB         // provides access to MongoDB
		conf             *config.Config       // contains details needed to access MongoDB
		log              *log.Logger          // main logger for RITA
		analyzedCallback func(mgoBulkActions) // analysis results are sent to this callback as MongoDB bulk actions
		closedCallback   func()               // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *uconn.Input    // holds unanalyzed unique connection data
		analysisWg       sync.WaitGroup       // wait for analysis to finish
	}
)

//newAnalyzer creates a new analyzer for calculating the beacon statistics of unique connections
func newAnalyzer(min int64, max int64, chunk int, db *database.DB, conf *config.Config, log *log.Logger,
	analyzedCallback func(mgoBulkActions), closedCallback func()) *analyzer {
	return &analyzer{
		tsMin:            min,
		tsMax:            max,
		chunk:            chunk,
		db:               db,
		conf:             conf,
		log:              log,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *uconn.Input),
	}
}

//collect gathers sorted unique connection data for analysis
func (a *analyzer) collect(data *uconn.Input) {
	a.analysisChannel <- data
}

//close waits for the analyzer to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

//start kicks off a new analysis thread
func (a *analyzer) start() {
	a.analysisWg.Add(1)
	go func() {

		for res := range a.analysisChannel {
			// if uconn has turned into a strobe, we will not have any timestamps here,
			// and need to update uconn table with the strobe flag. This is being done
			// here and not in uconns because uconns doesn't do reads, and doesn't know
			// the updated conn count
			if (res.TsList) == nil {
				// copy variables to be used by bulk callback to prevent capturing by reference
				pairSelector := res.Hosts.BSONKey()
				update := mgoBulkActions{
					a.conf.T.Structure.UniqueConnTable: func(b *mgo.Bulk) int {
						b.Upsert(
							pairSelector,
							bson.M{
								"$set": bson.M{"strobe": true},
							},
						)
						return 1
					},
					a.conf.T.Beacon.BeaconTable: func(b *mgo.Bulk) int {
						b.Remove(pairSelector)
						return 1
					},
				}
				a.analyzedCallback(update)
			} else {
				//store the diff slice length since we use it a lot
				//for timestamps this is one less then the data slice length
				//since we are calculating the times in between readings
				tsLength := len(res.TsList) - 1
				dsLength := len(res.OrigBytesList)

				//find the delta times between the timestamps
				diff := make([]int64, tsLength)
				for i := 0; i < tsLength; i++ {
					diff[i] = res.TsList[i+1] - res.TsList[i]
				}

				//find the delta times between full list of timestamps
				//(this will be used for the intervals list. Bowleys skew
				//must use a unique timestamp list with no duplicates)
				tsLengthFull := len(res.TsListFull) - 1
				//find the delta times between the timestamps
				diffFull := make([]int64, tsLengthFull)
				for i := 0; i < tsLengthFull; i++ {
					diffFull[i] = res.TsListFull[i+1] - res.TsListFull[i]
				}

				//perfect beacons should have symmetric delta time and size distributions
				//Bowley's measure of skew is used to check symmetry
				sort.Sort(util.SortableInt64(diff))
				tsSkew := float64(0)
				dsSkew := float64(0)

				//tsLength -1 is used since diff is a zero based slice
				tsLow := diff[util.Round(.25*float64(tsLength-1))]
				tsMid := diff[util.Round(.5*float64(tsLength-1))]
				tsHigh := diff[util.Round(.75*float64(tsLength-1))]
				tsBowleyNum := tsLow + tsHigh - 2*tsMid
				tsBowleyDen := tsHigh - tsLow

				//we do the same for datasizes
				dsLow := res.OrigBytesList[util.Round(.25*float64(dsLength-1))]
				dsMid := res.OrigBytesList[util.Round(.5*float64(dsLength-1))]
				dsHigh := res.OrigBytesList[util.Round(.75*float64(dsLength-1))]
				dsBowleyNum := dsLow + dsHigh - 2*dsMid
				dsBowleyDen := dsHigh - dsLow

				//tsSkew should equal zero if the denominator equals zero
				//bowley skew is unreliable if Q2 = Q1 or Q2 = Q3
				if tsBowleyDen != 0 && tsMid != tsLow && tsMid != tsHigh {
					tsSkew = float64(tsBowleyNum) / float64(tsBowleyDen)
				}

				if dsBowleyDen != 0 && dsMid != dsLow && dsMid != dsHigh {
					dsSkew = float64(dsBowleyNum) / float64(dsBowleyDen)
				}

				//perfect beacons should have very low dispersion around the
				//median of their delta times
				//Median Absolute Deviation About the Median
				//is used to check dispersion
				devs := make([]int64, tsLength)
				for i := 0; i < tsLength; i++ {
					devs[i] = util.Abs(diff[i] - tsMid)
				}

				dsDevs := make([]int64, dsLength)
				for i := 0; i < dsLength; i++ {
					dsDevs[i] = util.Abs(res.OrigBytesList[i] - dsMid)
				}

				sort.Sort(util.SortableInt64(devs))
				sort.Sort(util.SortableInt64(dsDevs))

				tsMadm := devs[util.Round(.5*float64(tsLength-1))]
				dsMadm := dsDevs[util.Round(.5*float64(dsLength-1))]

				//Store the range for human analysis
				tsIntervalRange := diff[tsLength-1] - diff[0]
				dsRange := res.OrigBytesList[dsLength-1] - res.OrigBytesList[0]

				//get a list of the intervals found in the data,
				//the number of times the interval was found,
				//and the most occurring interval
				//sort intervals list (origbytes already sorted)
				sort.Sort(util.SortableInt64(diffFull))
				intervals, intervalCounts, tsMode, tsModeCount := createCountMap(diffFull)
				dsSizes, dsCounts, dsMode, dsModeCount := createCountMap(res.OrigBytesList)

				//more skewed distributions receive a lower score
				//less skewed distributions receive a higher score
				tsSkewScore := 1.0 - math.Abs(tsSkew) //smush tsSkew
				dsSkewScore := 1.0 - math.Abs(dsSkew) //smush dsSkew

				//lower dispersion is better
				tsMadmScore := 1.0 - float64(tsMadm)/float64(tsMid)
				if tsMadmScore < 0 {
					tsMadmScore = 0
				}

				//lower dispersion is better
				dsMadmScore := 1.0 - float64(dsMadm)/float64(dsMid)
				if dsMadmScore < 0 {
					dsMadmScore = 0
				}

				//smaller data sizes receive a higher score
				dsSmallnessScore := 1.0 - float64(dsMode)/65535.0
				if dsSmallnessScore < 0 {
					dsSmallnessScore = 0
				}

				// connection count scoring
				tsConnDiv := (float64(a.tsMax) - float64(a.tsMin)) / 3600
				tsConnCountScore := float64(res.ConnectionCount) / tsConnDiv
				if tsConnCountScore > 1.0 {
					tsConnCountScore = 1.0
				}

				// calculate final ts and ds scores
				tsScore := math.Ceil(((tsSkewScore+tsMadmScore+tsConnCountScore)/3.0)*1000) / 1000
				dsScore := math.Ceil(((dsSkewScore+dsMadmScore+dsSmallnessScore)/3.0)*1000) / 1000

				// calculate duration score
				duration := math.Ceil((float64(res.TsList[tsLength]-res.TsList[0])/(float64(a.tsMax)-float64(a.tsMin)))*1000) / 1000
				if duration > 1.0 {
					duration = 1.0
				}

				// calculate histogram score
				bucketDivs, freqList, freqCount, histScore := getTsHistogramScore(a.tsMin, a.tsMax, res.TsList)

				// calculate overall beacon score
				score := math.Ceil(((tsScore*a.conf.S.Beacon.TsWeight)+
					(dsScore*a.conf.S.Beacon.DsWeight)+
					(duration*a.conf.S.Beacon.DurWeight)+
					(histScore*a.conf.S.Beacon.HistWeight))*1000) / 1000

				// copy variables to be used by bulk callback to prevent capturing by reference
				pairSelector := res.Hosts.BSONKey()
				beaconQuery := bson.M{
					"$set": bson.M{
						"connection_count":   res.ConnectionCount,
						"avg_bytes":          res.TotalBytes / res.ConnectionCount,
						"total_bytes":        res.TotalBytes,
						"ts.range":           tsIntervalRange,
						"ts.mode":            tsMode,
						"ts.mode_count":      tsModeCount,
						"ts.intervals":       intervals,
						"ts.interval_counts": intervalCounts,
						"ts.dispersion":      tsMadm,
						"ts.skew":            tsSkew,
						"ts.conns_score":     tsConnCountScore,
						"ts.score":           tsScore,
						"ds.range":           dsRange,
						"ds.mode":            dsMode,
						"ds.mode_count":      dsModeCount,
						"ds.sizes":           dsSizes,
						"ds.counts":          dsCounts,
						"ds.dispersion":      dsMadm,
						"ds.skew":            dsSkew,
						"ds.score":           dsScore,
						"duration_score":     duration,
						"bucket_divs":        bucketDivs,
						"freq_list":          freqList,
						"freq_count":         freqCount,
						"hist_score":         histScore,
						"score":              score,
						"cid":                a.chunk,
						"src_network_name":   res.Hosts.SrcNetworkName,
						"dst_network_name":   res.Hosts.DstNetworkName,
					},
				}

				update := mgoBulkActions{
					a.conf.T.Beacon.BeaconTable: func(b *mgo.Bulk) int {
						b.Upsert(pairSelector, beaconQuery)
						return 1
					},
				}

				a.analyzedCallback(update)
			}
		}
		a.analysisWg.Done()
	}()
}

// createCountMap returns a distinct data array, data count array, the mode,
// and the number of times the mode occurred
func createCountMap(sortedIn []int64) ([]int64, []int64, int64, int64) {
	//Since the data is already sorted, we can call this without fear
	distinct, countsMap := countAndRemoveConsecutiveDuplicates(sortedIn)
	countsArr := make([]int64, len(distinct))
	mode := distinct[0]
	max := countsMap[mode]
	for i, datum := range distinct {
		count := countsMap[datum]
		countsArr[i] = count
		if count > max {
			max = count
			mode = datum
		}
	}
	return distinct, countsArr, mode, max
}

//countAndRemoveConsecutiveDuplicates removes consecutive
//duplicates in an array of integers and counts how many
//instances of each number exist in the array.
//Similar to `uniq -c`, but counts all duplicates, not just
//consecutive duplicates.
func countAndRemoveConsecutiveDuplicates(numberList []int64) ([]int64, map[int64]int64) {
	//Avoid some reallocations
	result := make([]int64, 0, len(numberList)/2)
	counts := make(map[int64]int64)

	last := numberList[0]
	result = append(result, last)
	counts[last]++

	for idx := 1; idx < len(numberList); idx++ {
		if last != numberList[idx] {
			result = append(result, numberList[idx])
		}
		last = numberList[idx]
		counts[last]++
	}
	return result, counts
}

//getTsHistogramScore calculates two potential scores based on the histogram of connections for the
// host pair and takes the max of the two scores.
func getTsHistogramScore(min int64, max int64, tsList []int64) ([]int64, []int, map[int]int, float64) {

	// get bucket list
	// we currently look at a 24 hour period
	bucketDivs := createBuckets(min, max, 24)

	// use timestamps to get freqencies for buckets
	freqList, freqCount, freqCV := createHistogram(bucketDivs, tsList)

	// calculate first potential score
	// histograms with bigger flat sections will score higher, up to 4 flat sections
	// this will score well for graphs that have flat sections with a big distance between them,
	// i.e. a beacon that alternates between 1 and 5 connections per hour
	score1 := math.Ceil((float64(4)/float64(len(freqCount)))*1000) / 1000
	if score1 > 1.0 {
		score1 = 1.0
	}

	// calculate second potential score
	// coefficient of variation will help score histograms that have jitter in the number of
	// connections but where the overall graph would still look relatively flat and consistent
	score2 := math.Ceil((1.0-float64(freqCV))*1000) / 1000
	if score2 > 1.0 {
		score2 = 1.0
	}

	return bucketDivs, freqList, freqCount, math.Max(score1, score2)

}

//createBuckets
func createBuckets(min int64, max int64, size int64) []int64 {
	// Set number of dividers. Since the dividers include the endpoints,
	// number of dividers will be one more than the number of desired buckets
	total := size + 1

	// declare list
	bucketDivs := make([]int64, total)

	// calculate step size
	step := (max - min) / (total - 1)

	// set first bucket value to min timestamp
	bucketDivs[0] = min

	// create evenly spaced timestamp buckets
	for i := int64(1); i < total; i++ {
		bucketDivs[i] = min + (i * step)
	}

	// set first bucket value to max timestamp
	bucketDivs[total-1] = max

	return bucketDivs
}

//createHistogram
func createHistogram(bucketDivs []int64, tsList []int64) ([]int, map[int]int, float64) {
	i := 0
	bucket := bucketDivs[i+1]

	// calculate the number of connections that occurred within the time span represented
	// by each bucket
	freqList := make([]int, len(bucketDivs)-1)

	// loop over sorted timestamp list
	for _, entry := range tsList {

		// increment if still in the current bucket
		if entry < bucket {
			freqList[i]++
			continue
		}

		// find the next bucket this value will fall under
		for j := i + 1; j < len(bucketDivs)-1; j++ {
			if entry < bucketDivs[j+1] {
				i = j
				bucket = bucketDivs[j+1]
				break
			}
		}

		// increment count
		// this will also capture and increment for a situation where the final timestamp is
		// equal to the final bucket
		freqList[i]++
	}

	// make a fequency count map to track how often each value in freqList appears
	freqCount := make(map[int]int)
	total := 0

	for _, item := range freqList {
		total += item
		if _, ok := freqCount[item]; !ok {
			freqCount[item] = 1
		} else {
			freqCount[item]++
		}
	}

	freqMean := float64(total) / float64(len(freqList))

	// calculate standard deviation
	sd := float64(0)
	for j := 0; j < len(freqList); j++ {
		sd += math.Pow(float64(freqList[j])-freqMean, 2)
	}
	sd = math.Sqrt(sd / float64(len(freqList)))

	// calculate coefficient of variation
	cv := sd / freqMean

	// if cv is greater than 1, our score should be zero
	if cv > 1.0 {
		cv = 1.0
	}

	return freqList, freqCount, cv

}
