package beacon

import (
	"math"
	"sort"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/parser/uconn"
	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo/bson"
)

type (
	analyzer struct {
		db               *database.DB    // provides access to MongoDB
		conf             *config.Config  // contains details needed to access MongoDB
		minTime          int64           // beginning of the observation period
		maxTime          int64           // ending of the observation period
		analyzedCallback func(update)    // called on each analyzed result
		closedCallback   func()          // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan uconn.Pair // holds unanalyzed data
		analysisWg       sync.WaitGroup  // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for gathering data
func newAnalyzer(db *database.DB, conf *config.Config, minTime, maxTime int64, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		db:               db,
		conf:             conf,
		minTime:          minTime,
		maxTime:          maxTime,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan uconn.Pair),
	}
}

//collect sends a chunk of data to be analyzed
func (a *analyzer) collect(data uconn.Pair) {
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

		for data := range a.analysisChannel {

			// This will work for both updating and inserting completely new Beacons
			// for every new uconn record we have, we will check the uconns table. This
			// will always return a result because even with a brand new database, we already
			// created the uconns table. It will only continue and analyze if the connection
			// meets the required specs, again working for both an update and a new src-dst pair.
			// We would have to perform this check regardless if we want the rolling update
			// option to remain, and this gets us the vetting for both situations, and Only
			// works on the current entries - not a re-aggregation on the whole collection,
			// and individual lookups like this are really fast. This also ensures a unique
			// set of timestamps for analysis.
			uconnFindQuery := bson.M{
				"$and": []bson.M{
					bson.M{"src": data.Src},
					bson.M{"dst": data.Dst},
					bson.M{"connection_count": bson.M{"$gt": a.conf.S.Beacon.DefaultConnectionThresh}},
					bson.M{"connection_count": bson.M{"$lt": 150000}},
					bson.M{"ts_list.4": bson.M{"$exists": true}},
				}}

			var res uconnRes

			_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.UniqueConnTable).Find(uconnFindQuery).Limit(1).One(&res)

			// Check for errors and parse results
			// this is here because it will still return an empty document even if there are no results
			// this verifies that. There's probably a better way, but i'm having a brain fart. I think we normally
			// return an iterator and go over results (won't enter loop if none), but those are way less efficient if you
			// are expecting only one entry, I read a thingie about it.
			if len(res.TsList) > 0 {

				//sort the size and timestamps since they may have arrived out of order
				sort.Sort(util.SortableInt64(res.TsList))
				sort.Sort(util.SortableInt64(res.OrigIPBytes))

				//store the diff slice length since we use it a lot
				//for timestamps this is one less then the data slice length
				//since we are calculating the times in between readings
				tsLength := len(res.TsList) - 1
				dsLength := len(res.OrigIPBytes)

				//find the duration of this connection
				//perfect beacons should fill the observation period
				duration := float64(res.TsList[tsLength]-res.TsList[0]) /
					float64(a.maxTime-a.minTime)

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
				dsLow := res.OrigIPBytes[util.Round(.25*float64(dsLength-1))]
				dsMid := res.OrigIPBytes[util.Round(.5*float64(dsLength-1))]
				dsHigh := res.OrigIPBytes[util.Round(.75*float64(dsLength-1))]
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
					dsDevs[i] = util.Abs(res.OrigIPBytes[i] - dsMid)
				}

				sort.Sort(util.SortableInt64(devs))
				sort.Sort(util.SortableInt64(dsDevs))

				tsMadm := devs[util.Round(.5*float64(tsLength-1))]
				dsMadm := dsDevs[util.Round(.5*float64(dsLength-1))]

				//Store the range for human analysis
				tsIntervalRange := diff[tsLength-1] - diff[0]
				dsRange := res.OrigIPBytes[dsLength-1] - res.OrigIPBytes[0]

				//get a list of the intervals found in the data,
				//the number of times the interval was found,
				//and the most occurring interval
				intervals, intervalCounts, tsMode, tsModeCount := createCountMap(diff)
				dsSizes, dsCounts, dsMode, dsModeCount := createCountMap(res.OrigIPBytes)

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

				tsDurationScore := duration

				//smaller data sizes receive a higher score
				dsSmallnessScore := 1.0 - float64(dsMode)/65535.0
				if dsSmallnessScore < 0 {
					dsSmallnessScore = 0
				}

				// output := &beacon.AnalysisOutput{
				// 	UconnID:          data.ID,
				// 	Src:              data.Src,
				// 	Dst:              data.Dst,
				// 	ConnectionCount:  data.ConnectionCount,
				// 	AverageBytes:     data.AverageBytes,
				// 	TSISkew:          tsSkew,
				// 	TSIDispersion:    tsMadm,
				// 	TSDuration:       duration,
				// 	TSIRange:         tsIntervalRange,
				// 	TSIMode:          tsMode,
				// 	TSIModeCount:     tsModeCount,
				// 	TSIntervals:      intervals,
				// 	TSIntervalCounts: intervalCounts,
				// 	DSSkew:           dsSkew,
				// 	DSDispersion:     dsMadm,
				// 	DSRange:          dsRange,
				// 	DSSizes:          dsSizes,
				// 	DSSizeCounts:     dsCounts,
				// 	DSMode:           dsMode,
				// 	DSModeCount:      dsModeCount,
				// }

				//score numerators
				tsSum := tsSkewScore + tsMadmScore + tsDurationScore
				dsSum := dsSkewScore + dsMadmScore + dsSmallnessScore

				//score averages
				tsScore := tsSum / 3.0
				dsScore := dsSum / 3.0
				score := (tsSum + dsSum) / 6.0

				// update beacon
				output := update{
					// create query
					query: bson.M{
						"$set": bson.M{
							"connection_count":   res.ConnectionCount,
							"avg_bytes":          res.AverageBytes,
							"ts_iRange":          tsIntervalRange,
							"ts_iMode":           tsMode,
							"ts_iMode_count":     tsModeCount,
							"ts_intervals":       intervals,
							"ts_interval_counts": intervalCounts,
							"ts_iDispersion":     tsMadm,
							"ts_iSkew":           tsSkew,
							"ts_duration":        duration,
							"ts_score":           tsScore,
							"ds_range":           dsRange,
							"ds_mode":            dsMode,
							"ds_mode_count":      dsModeCount,
							"ds_sizes":           dsSizes,
							"ds_counts":          dsCounts,
							"ds_dispersion":      dsMadm,
							"ds_skew":            dsSkew,
							"ds_score":           dsScore,
							"score":              score,
						},
					},
					// create selector for output
					selector: bson.M{"src": data.Src, "dst": data.Dst},
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
