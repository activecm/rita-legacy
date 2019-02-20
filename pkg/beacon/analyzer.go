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
)

type (
	analyzer struct {
		db               *database.DB     // provides access to MongoDB
		conf             *config.Config   // contains details needed to access MongoDB
		analyzedCallback func(*update)    // called on each analyzed result
		closedCallback   func()           // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *uconn.Pair // holds unanalyzed data
		analysisWg       sync.WaitGroup   // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for gathering data
func newAnalyzer(db *database.DB, conf *config.Config, analyzedCallback func(*update), closedCallback func()) *analyzer {
	return &analyzer{
		db:               db,
		conf:             conf,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *uconn.Pair),
	}
}

//collect sends a chunk of data to be analyzed
func (a *analyzer) collect(data *uconn.Pair) {
	a.analysisChannel <- data
}

//close waits for the collector to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

//start kicks off a new analysis thread
func (a *analyzer) start() {
	a.analysisWg.Add(1)
	go func() {
		ssn := a.db.Session.Copy()
		defer ssn.Close()

		for res := range a.analysisChannel {

			output := &update{}

			// if uconn has turned into a strobe, we will not have any timestamps here,
			// and need to update uconn table with the strobe flag. This is being done
			// here and not in uconns because uconns doesn't do reads, and doesn't know
			// the updated conn count
			if (res.TsList) == nil {
				output.uconn = updateInfo{
					// update hosts record
					query: bson.M{
						"$set": bson.M{"strobe": true},
					},
					// create selector for output
					selector: bson.M{"src": res.Src, "dst": res.Dst},
				}

				// set to writer channel
				a.analyzedCallback(output)

			} else {

				//sort the size and timestamps since they may have arrived out of order
				sort.Sort(util.SortableInt64(res.TsList))
				sort.Sort(util.SortableInt64(res.OrigBytesList))

				//remove subsecond communications
				//these will appear as beacons if we do not remove them
				res.TsList = removeConsecutiveDuplicates(res.TsList)

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
				intervals, intervalCounts, tsMode, tsModeCount := createCountMap(diff)
				dsSizes, dsCounts, dsMode, dsModeCount := createCountMap(res.OrigBytesList)

				//more skewed distributions receive a lower score
				//less skewed distributions receive a higher score
				tsSkewScore := 1.0 - math.Abs(tsSkew) //smush tsSkew
				dsSkewScore := 1.0 - math.Abs(dsSkew) //smush dsSkew

				//lower dispersion is better, cutoff dispersion scores at 30 seconds
				tsMadmScore := 1.0 - float64(tsMadm)/30.0
				if tsMadmScore < 0 {
					tsMadmScore = 0
				}

				//lower dispersion is better, cutoff dispersion scores at 32 bytes
				dsMadmScore := 1.0 - float64(dsMadm)/32.0
				if dsMadmScore < 0 {
					dsMadmScore = 0
				}

				//smaller data sizes receive a higher score
				dsSmallnessScore := 1.0 - float64(dsMode)/65535.0
				if dsSmallnessScore < 0 {
					dsSmallnessScore = 0
				}

				//score numerators
				tsSum := tsSkewScore + tsMadmScore
				dsSum := dsSkewScore + dsMadmScore + dsSmallnessScore

				//score averages
				tsScore := math.Ceil((tsSum/2.0)*1000) / 1000
				dsScore := math.Ceil((dsSum/3.0)*1000) / 1000
				score := math.Ceil(((tsSum+dsSum)/5.0)*1000) / 1000

				// update beacon query
				output.beacon = updateInfo{
					query: bson.M{
						"$set": bson.M{
							"src":                res.Src,
							"dst":                res.Dst,
							"connection_count":   res.ConnectionCount,
							"avg_bytes":          res.TotalBytes / res.ConnectionCount,
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
							"score":              score,
						},
					},
					selector: bson.M{"src": res.Src, "dst": res.Dst},
				}

				output.host = updateInfo{
					// update hosts record
					query: bson.M{
						"$max": bson.M{"dat.0.max_beacon_score": score},
					},
					// create selector for output
					selector: bson.M{"ip": res.Src},
				}

				// set to writer channel
				a.analyzedCallback(output)

			}

		}
		a.analysisWg.Done()
	}()
}

// createCountMap returns a distinct data array, data count array, the mode,
// and the number of times the mode occured
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

//CountAndRemoveConsecutiveDuplicates removes consecutive
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

//removeConsecutiveDuplicates removes consecutive duplicates
func removeConsecutiveDuplicates(numberList []int64) []int64 {
	//Avoid some reallocations
	result := make([]int64, 0, len(numberList)/2)
	last := numberList[0]
	result = append(result, last)

	for idx := 1; idx < len(numberList); idx++ {
		if last != numberList[idx] {
			result = append(result, numberList[idx])
		}
		last = numberList[idx]
	}
	return result

}
