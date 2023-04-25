package beacon

import (
	"math"
	"sort"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/util"

	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type (
	//analyzer handles calculating statistical measures of the distributions of the
	//timestamps and data sizes between pairs of hosts
	analyzer struct {
		tsMin            int64                      // min timestamp for the whole dataset
		tsMax            int64                      // max timestamp for the whole dataset
		chunk            int                        // current chunk (0 if not on rolling analysis)
		db               *database.DB               // provides access to MongoDB
		conf             *config.Config             // contains details needed to access MongoDB
		log              *log.Logger                // main logger for RITA
		analyzedCallback func(database.BulkChanges) // analysis results are sent to this callback as MongoDB bulk actions
		closedCallback   func()                     // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *uconn.Input          // holds unanalyzed unique connection data
		analysisWg       sync.WaitGroup             // wait for analysis to finish
	}
)

// newAnalyzer creates a new analyzer for calculating the beacon statistics of unique connections
func newAnalyzer(min int64, max int64, chunk int, db *database.DB, conf *config.Config, log *log.Logger,
	analyzedCallback func(database.BulkChanges), closedCallback func()) *analyzer {
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

// collect gathers sorted unique connection data for analysis
func (a *analyzer) collect(data *uconn.Input) {
	a.analysisChannel <- data
}

// close waits for the analyzer to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

// start kicks off a new analysis thread
func (a *analyzer) start() {
	a.analysisWg.Add(1)
	go func() {
		for res := range a.analysisChannel {

			//store the diffFull slice length since we use it a lot
			//for timestamps this is one less then the data slice length
			//since we are calculating the times in between readings
			tsLength := len(res.TsList) - 1
			dsLength := len(res.OrigBytesList)

			//find the delta times between the timestamps and sort
			diffFull := make([]int64, tsLength)
			for i := 0; i < tsLength; i++ {
				interval := res.TsList[i+1] - res.TsList[i]
				diffFull[i] = interval
			}
			sort.Sort(util.SortableInt64(diffFull))

			// We are excluding delta zero for scoring calculations
			// but using a separate array that includes it for making
			// the user/ graph reference variables returned by createCountMap.

			// Search for the section of diffFull without any 0's in it
			// The dissector guarantees that there are at least three unique timestamps in res.TsList
			// as a result, we are guaranteed to find at least two non-zero intervals in diffFull
			diffNonZeroIdx := 0
			for i := 0; i < len(diffFull); i++ {
				if diffFull[i] > 0 {
					diffNonZeroIdx = i
					break
				}
			}

			diff := diffFull[diffNonZeroIdx:] // select the part of diffFull without any 0's

			//store the diff slice length
			diffLength := len(diff)

			//perfect beacons should have symmetric delta time and size distributions
			//Bowley's measure of skew is used to check symmetry
			tsSkew := float64(0)
			dsSkew := float64(0)

			//diffLength-1 is used since diff is a zero based slice
			tsLow := diff[util.Round(.25*float64(diffLength-1))]
			tsMid := diff[util.Round(.5*float64(diffLength-1))]
			tsHigh := diff[util.Round(.75*float64(diffLength-1))]
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
			if tsBowleyDen >= 10 && tsMid != tsLow && tsMid != tsHigh {
				tsSkew = float64(tsBowleyNum) / float64(tsBowleyDen)
			}

			if dsBowleyDen >= 10 && dsMid != dsLow && dsMid != dsHigh {
				dsSkew = float64(dsBowleyNum) / float64(dsBowleyDen)
			}

			//perfect beacons should have very low dispersion around the
			//median of their delta times
			//Median Absolute Deviation About the Median
			//is used to check dispersion
			devs := make([]int64, diffLength)
			for i := 0; i < diffLength; i++ {
				devs[i] = util.Abs(diff[i] - tsMid)
			}

			dsDevs := make([]int64, dsLength)
			for i := 0; i < dsLength; i++ {
				dsDevs[i] = util.Abs(res.OrigBytesList[i] - dsMid)
			}

			sort.Sort(util.SortableInt64(devs))
			sort.Sort(util.SortableInt64(dsDevs))

			tsMadm := devs[util.Round(.5*float64(diffLength-1))]
			dsMadm := dsDevs[util.Round(.5*float64(dsLength-1))]

			//Store the range for human analysis
			tsIntervalRange := diff[diffLength-1] - diff[0]
			dsRange := res.OrigBytesList[dsLength-1] - res.OrigBytesList[0]

			//get a list of the intervals found in the data,
			//the number of times the interval was found,
			//and the most occurring interval
			intervals, intervalCounts, tsMode, tsModeCount := createCountMap(diffFull)
			dsSizes, dsCounts, dsMode, dsModeCount := createCountMap(res.OrigBytesList)

			//more skewed distributions receive a lower score
			//less skewed distributions receive a higher score
			tsSkewScore := 1.0 - math.Abs(tsSkew) //smush tsSkew
			dsSkewScore := 1.0 - math.Abs(dsSkew) //smush dsSkew

			//lower dispersion is better
			tsMadmScore := 1.0
			if tsMid >= 1 {
				tsMadmScore = 1.0 - float64(tsMadm)/float64(tsMid)
			}
			if tsMadmScore < 0 {
				tsMadmScore = 0
			}

			//lower dispersion is better
			dsMadmScore := 0.0
			if dsMid >= 1 {
				dsMadmScore = 1.0 - float64(dsMadm)/float64(dsMid)
			}
			if dsMadmScore < 0 {
				dsMadmScore = 0
			}

			//smaller data sizes receive a higher score
			dsSmallnessScore := 1.0 - float64(dsMode)/65535.0
			if dsSmallnessScore < 0 {
				dsSmallnessScore = 0
			}

			// calculate final ts and ds scores
			tsScore := math.Ceil(((tsSkewScore+tsMadmScore)/2.0)*1000) / 1000
			dsScore := math.Ceil(((dsSkewScore+dsMadmScore+dsSmallnessScore)/3.0)*1000) / 1000

			// calculate histogram score
			bucketDivs, freqList, freqCount, totalBars, longestRun, histScore := getTsHistogramScore(a.tsMin, a.tsMax, res.TsList, a.conf.S.Beacon.HistBimodalBucketSize, a.conf.S.Beacon.HistBimodalOutlierRemoval, a.conf.S.Beacon.HistBimodalMinHoursSeen)

			// calculate duration score
			durScore := getDurationScore(a.tsMin, a.tsMax, res.TsList[0], res.TsList[tsLength], totalBars, longestRun, a.conf.S.Beacon.DurMinHoursSeen, a.conf.S.Beacon.DurConsistencyIdealHoursSeen)

			// calculate overall beacon score
			score := math.Ceil(((tsScore*a.conf.S.Beacon.TsWeight)+
				(dsScore*a.conf.S.Beacon.DsWeight)+
				(durScore*a.conf.S.Beacon.DurWeight)+
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
					"ts.score":           tsScore,
					"ds.range":           dsRange,
					"ds.mode":            dsMode,
					"ds.mode_count":      dsModeCount,
					"ds.sizes":           dsSizes,
					"ds.counts":          dsCounts,
					"ds.dispersion":      dsMadm,
					"ds.skew":            dsSkew,
					"ds.score":           dsScore,
					"duration_score":     durScore,
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

			update := database.BulkChanges{
				a.conf.T.Beacon.BeaconTable: []database.BulkChange{
					{Selector: pairSelector, Update: beaconQuery, Upsert: true},
				},
			}

			a.analyzedCallback(update)
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

// countAndRemoveConsecutiveDuplicates removes consecutive
// duplicates in an array of integers and counts how many
// instances of each number exist in the array.
// Similar to `uniq -c`, but counts all duplicates, not just
// consecutive duplicates.
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

// getTsHistogramScore calculates two potential scores based on the histogram of connections for the
// host pair and takes the max of the two scores.
func getTsHistogramScore(min int64, max int64, tsList []int64, bimodalBucketSize float64, bimodalOutlierRemoval int, bimodalMinHoursSeen int) ([]int64, []int, map[int]int, int, int, float64) {

	// get bucket list
	// we currently look at a 24 hour period
	bucketDivs := createBuckets(min, max, 24)

	// use timestamps to get freqencies for buckets
	freqList, freqCount, total, totalBars, longestRun := createHistogram(bucketDivs, tsList, bimodalBucketSize)

	// calculate first potential score
	// coefficient of variation will help score histograms that have jitter in the number of
	// connections but where the overall graph would still look relatively flat and consistent

	// calculate mean
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

	cvScore := math.Ceil((1.0-float64(cv))*1000) / 1000
	if cvScore > 1.0 {
		cvScore = 1.0
	}

	// Calculate second potential score
	// this will score well for graphs that have 2-3 flat sections in their connection histogram,
	// or a bimodal freqCount histogram.
	// Example - a beacon that alternates between 1 and 5 connections per hour
	// This score will only be calculated if the number of total bars on the histogram is at
	// least the amount set in the yaml file (default: 11)
	bimodalFit := float64(0)

	if totalBars >= bimodalMinHoursSeen {
		largest := 0
		secondLargest := 0

		// get top two frequency mode bars
		for _, value := range freqCount {
			if value > largest {
				secondLargest = largest
				largest = value
			} else if value > secondLargest {
				secondLargest = value
			}
		}

		// calculate the percentage of hour blocks that fit into the top two mode buckets.
		// a small buffer for the score is provided by throwing out a yaml-set number of
		// potential outlier buckets (default: 1)
		bimodalFit = float64(largest+secondLargest) / float64(util.Max(totalBars-bimodalOutlierRemoval, 1))
	}

	bimodalFitScore := math.Ceil((float64(bimodalFit))*1000) / 1000
	if bimodalFitScore > 1.0 {
		bimodalFitScore = 1.0
	}

	return bucketDivs, freqList, freqCount, totalBars, longestRun, math.Max(cvScore, bimodalFitScore)

}

// createBuckets
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

// createHistogram
func createHistogram(bucketDivs []int64, tsList []int64, bimodalBucketSize float64) ([]int, map[int]int, int, int, int) {
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

	// get histogram frequency counts
	freqCount, total, totalBars, longestRun := getFrequencyCounts(freqList, bimodalBucketSize)

	return freqList, freqCount, total, totalBars, longestRun

}

func getFrequencyCounts(freqList []int, bimodalBucketSize float64) (map[int]int, int, int, int) {

	// count total non-zero histogram entries (total bars) and find the
	// largest histogram entry
	totalBars := 0
	largestConnCount := 0
	for _, entry := range freqList {
		if entry > 0 {
			totalBars++
		}
		if entry > largestConnCount {
			largestConnCount = entry
		}

	}

	// make a fequency count map to track how often each value in freqList appears
	freqCount := make(map[int]int)
	total := 0

	// determine bucket size for frequency histogram. This is expressed as a percentage of the
	// largest connection count and controls how forgiving the bimodal analysis is to variation.
	// the percentage is set in the rita yaml file (default: 0.05)
	bucketSize := math.Ceil(float64(largestConnCount) * bimodalBucketSize)

	// make variables to track the longest consecutive run of hours seen in the connection
	// frequency histogram, including wrap around from start to end of dataset
	freqListLen := len(freqList)
	longestRun := 0
	currentRun := 0

	// make frequency count map
	for i := 0; i < freqListLen*2; i++ {

		item := freqList[i%freqListLen]

		if item > 0 {
			currentRun++

		} else {

			if currentRun > longestRun {
				longestRun = currentRun
			}
			currentRun = 0

		}

		if i < freqListLen {
			total += item

			// exclude zero-valued entries
			if item > 0 {

				// figure out which bucket to parse the frequency bar into
				bucket := int(math.Floor(float64(item)/bucketSize) * bucketSize)

				// create or increment bucket
				if _, ok := freqCount[bucket]; !ok {
					freqCount[bucket] = 1
				} else {
					freqCount[bucket]++
				}
			}

		}

	}

	if currentRun > longestRun {
		longestRun = currentRun
	}

	// since we could end up with 2*freqListLen for the longest run if
	// every hour has a connection, we will fix it up here.
	if longestRun > freqListLen {
		longestRun = freqListLen
	}

	return freqCount, total, totalBars, longestRun
}

// getDurationScore
func getDurationScore(min int64, max int64, tsListMin int64, tsListMax int64, totalBars int, longestRun int, minHoursSeen, consistencyIdealHoursSeen int) float64 {
	// Duration will only be calculated if more than the yaml-defined  threshold (default: 6) hours are
	// represented in the connection frequency histogram
	// Duration Score will take the maximum of two potential subscores:
	// Dataset Timespan Coverage
	// [ timestamp of last connection - timestamp of first connection ] /
	// [ last timestamp of dataset - first timestamp of dataset ]
	// Consistency
	// [ longest run of consecutive hours seen] / [ 12 hours* ]
	// note: consecutive includes wrap around from start to end of dataset
	// *ideal number of consecutive hours can be adjusted in the rita yaml file (default: 12)

	durScore := 0.0

	if totalBars > minHoursSeen {

		coverageScore := math.Ceil((float64(tsListMax-tsListMin)/(float64(max)-float64(min)))*1000) / 1000
		if coverageScore > 1.0 {
			coverageScore = 1.0
		}

		consistencyScore := math.Ceil((float64(longestRun)/float64(consistencyIdealHoursSeen))*1000) / 1000
		if consistencyScore > 1.0 {
			consistencyScore = 1.0
		}

		durScore = math.Max(coverageScore, consistencyScore)
	}

	return durScore
}
