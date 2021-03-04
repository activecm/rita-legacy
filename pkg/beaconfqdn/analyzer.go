package beaconfqdn

import (
	"math"
	"sort"
	"strconv"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo/bson"
)

type (
	analyzer struct {
		tsMin            int64                    // min timestamp for the whole dataset
		tsMax            int64                    // max timestamp for the whole dataset
		chunk            int                      //current chunk (0 if not on rolling analysis)
		chunkStr         string                   //current chunk (0 if not on rolling analysis)
		db               *database.DB             // provides access to MongoDB
		conf             *config.Config           // contains details needed to access MongoDB
		analyzedCallback func(*update)            // called on each analyzed result
		closedCallback   func()                   // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *hostname.FqdnInput // holds unanalyzed data
		analysisWg       sync.WaitGroup           // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for gathering data //
func newAnalyzer(min int64, max int64, chunk int, db *database.DB, conf *config.Config, analyzedCallback func(*update), closedCallback func()) *analyzer {
	return &analyzer{
		tsMin:            min,
		tsMax:            max,
		chunk:            chunk,
		chunkStr:         strconv.Itoa(chunk),
		db:               db,
		conf:             conf,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *hostname.FqdnInput),
	}
}

//collect sends a chunk of data to be analyzed
func (a *analyzer) collect(data *hostname.FqdnInput) {
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

		for entry := range a.analysisChannel {

			// set up beacon writer output
			output := &update{}

			// create selector pair object
			selectorPair := uniqueSrcHostnamePair{
				entry.Src.SrcIP,
				entry.Src.SrcNetworkUUID,
				entry.FQDN,
			}

			// create query
			query := bson.M{}

			// if beacon has turned into a strobe, we will not have any timestamps here,
			// and need to update beaconFQDN table with the strobeFQDN flag.
			if (entry.TsList) == nil {

				// set strobe info
				query["$set"] = bson.M{
					"strobeFQDN":       true,
					"total_bytes":      entry.TotalBytes,
					"avg_bytes":        entry.TotalBytes / entry.ConnectionCount,
					"connection_count": entry.ConnectionCount,
					"src_network_name": entry.Src.SrcNetworkName,
					"resolved_ips":     entry.ResolvedIPs,
					"cid":              a.chunk,
				}

				// unset any beacon calculations since  this
				// is now a strobe and those would be inaccurate
				// (this will only apply to chunked imports)
				query["$unset"] = bson.M{
					"ts":    1,
					"ds":    1,
					"score": 1,
				}

				// create selector for output
				output.beacon.query = query
				output.beacon.selector = selectorPair.BSONKey()

				// set to writer channel
				a.analyzedCallback(output)

			} else {
				//store the diff slice length since we use it a lot
				//for timestamps this is one less then the data slice length
				//since we are calculating the times in between readings
				tsLength := len(entry.TsList) - 1
				dsLength := len(entry.OrigBytesList)

				//find the delta times between the timestamps
				diff := make([]int64, tsLength)
				for i := 0; i < tsLength; i++ {
					diff[i] = entry.TsList[i+1] - entry.TsList[i]
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
				dsLow := entry.OrigBytesList[util.Round(.25*float64(dsLength-1))]
				dsMid := entry.OrigBytesList[util.Round(.5*float64(dsLength-1))]
				dsHigh := entry.OrigBytesList[util.Round(.75*float64(dsLength-1))]
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
					dsDevs[i] = util.Abs(entry.OrigBytesList[i] - dsMid)
				}

				sort.Sort(util.SortableInt64(devs))
				sort.Sort(util.SortableInt64(dsDevs))

				tsMadm := devs[util.Round(.5*float64(tsLength-1))]
				dsMadm := dsDevs[util.Round(.5*float64(dsLength-1))]

				//Store the range for human analysis
				tsIntervalRange := diff[tsLength-1] - diff[0]
				dsRange := entry.OrigBytesList[dsLength-1] - entry.OrigBytesList[0]

				//get a list of the intervals found in the data,
				//the number of times the interval was found,
				//and the most occurring interval
				intervals, intervalCounts, tsMode, tsModeCount := createCountMap(diff)
				dsSizes, dsCounts, dsMode, dsModeCount := createCountMap(entry.OrigBytesList)

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

				// connection count scoring
				tsConnDiv := (float64(a.tsMax) - float64(a.tsMin)) / 10.0
				tsConnCountScore := float64(entry.ConnectionCount) / tsConnDiv
				if tsConnCountScore > 1.0 {
					tsConnCountScore = 1.0
				}

				//score numerators
				tsSum := tsSkewScore + tsMadmScore + tsConnCountScore
				dsSum := dsSkewScore + dsMadmScore + dsSmallnessScore

				//score averages
				tsScore := math.Ceil((tsSum/3.0)*1000) / 1000
				dsScore := math.Ceil((dsSum/3.0)*1000) / 1000
				score := math.Ceil(((tsSum+dsSum)/6.0)*1000) / 1000

				// update beacon query
				query["$set"] = bson.M{
					"connection_count":   entry.ConnectionCount,
					"avg_bytes":          entry.TotalBytes / entry.ConnectionCount,
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
					"score":              score,
					"cid":                a.chunk,
					"src_network_name":   entry.Src.SrcNetworkName,
					"resolved_ips":       entry.ResolvedIPs,
				}

				// set query
				output.beacon.query = query

				// create selector for output
				output.beacon.selector = selectorPair.BSONKey()

				// updates source entry in hosts table to flag for invalid certificates
				output.hostIcert = a.hostIcertQuery(entry.InvalidCertFlag, entry.Src, entry.FQDN)

				// updates max FQDN beacon score for the source entry in the hosts table
				output.hostBeacon = a.hostBeaconQuery(score, entry.Src, entry.FQDN)

				// set to writer channel
				a.analyzedCallback(output)
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

func (a *analyzer) hostIcertQuery(icert bool, src data.UniqueSrcIP, fqdn string) updateInfo {
	ssn := a.db.Session.Copy()
	defer ssn.Close()

	var output updateInfo

	// create query
	query := bson.M{}

	// update host table if there is an invalid cert record between pair
	if icert == true {

		newFlag := false

		var resList []interface{}

		hostSelector := src.SrcBSONKey()
		hostSelector["dat.icfqdn"] = fqdn

		_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(hostSelector).All(&resList)

		if len(resList) <= 0 {
			newFlag = true
		}

		if newFlag {

			query["$push"] = bson.M{
				"dat": bson.M{
					"icfqdn": fqdn,
					"icert":  1,
					"cid":    a.chunk,
				}}

			// create selector for output
			output.query = query
			output.selector = src.SrcBSONKey()

		} else {

			query["$set"] = bson.M{
				"dat.$.icert": 1,
				"dat.$.cid":   a.chunk,
			}

			// create selector for output
			output.query = query
			output.selector = hostSelector
		}
	}

	return output
}

func (a *analyzer) hostBeaconQuery(score float64, src data.UniqueSrcIP, fqdn string) updateInfo {
	ssn := a.db.Session.Copy()
	defer ssn.Close()

	var output updateInfo

	// create query
	query := bson.M{}

	// check if we need to update
	// we do this before the other queries because otherwise if a beacon
	// starts out with a high score which reduces over time, it will keep
	// the incorrect high max for that specific destination.
	var resListExactMatch []interface{}

	maxBeaconMatchExactQuery := src.SrcBSONKey()
	maxBeaconMatchExactQuery["dat.mbfqdn"] = fqdn

	_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(maxBeaconMatchExactQuery).All(&resListExactMatch)

	// if we have exact matches, update to new score and return
	if len(resListExactMatch) > 0 {
		query["$set"] = bson.M{
			"dat.$.max_beacon_fqdn_score": score,
			"dat.$.mbfqdn":                fqdn,
			"dat.$.cid":                   a.chunk,
		}

		// create selector for output
		output.query = query

		// using the same find query we created above will allow us to match and
		// update the exact chunk we need to update
		output.selector = maxBeaconMatchExactQuery

		return output
	}

	// The below is only for cases where the ip is not currently listed as a max beacon
	// for a source
	// update max beacon score
	newFlag := false
	updateFlag := false

	// this query will find any matching chunk that is reporting a lower
	// max beacon score than the current one we are working with
	maxBeaconMatchLowerQuery := src.SrcBSONKey()
	maxBeaconMatchLowerQuery["dat"] = bson.M{
		"$elemMatch": bson.M{
			"cid":                   a.chunk,
			"max_beacon_fqdn_score": bson.M{"$lte": score},
		},
	}
	// find matching lower chunks
	var resListLower []interface{}

	_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(maxBeaconMatchLowerQuery).All(&resListLower)

	// if no matching chunks are found, we will set the new flag
	if len(resListLower) <= 0 {

		maxBeaconMatchUpperQuery := src.SrcBSONKey()
		maxBeaconMatchUpperQuery["dat"] = bson.M{
			"$elemMatch": bson.M{
				"cid":                   a.chunk,
				"max_beacon_fqdn_score": bson.M{"$gte": score},
			},
		}

		// find matching upper chunks
		var resListUpper []interface{}
		_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(maxBeaconMatchUpperQuery).All(&resListUpper)
		// update if no upper chunks are found
		if len(resListUpper) <= 0 {
			newFlag = true
		}
	} else {
		updateFlag = true
	}

	// since we didn't find any changeable lower max beacon scores, we will
	// set the condition to push a new entry with the current score listed as the
	// max beacon ONLY if no matching chunks reporting higher max beacon scores
	// are found.

	if newFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"max_beacon_fqdn_score": score,
				"mbfqdn":                fqdn,
				"cid":                   a.chunk,
			}}

		// create selector for output
		output.query = query
		output.selector = src.SrcBSONKey()

	} else if updateFlag {

		query["$set"] = bson.M{
			"dat.$.max_beacon_fqdn_score": score,
			"dat.$.mbfqdn":                fqdn,
			"dat.$.cid":                   a.chunk,
		}

		// create selector for output
		output.query = query

		// using the same find query we created above will allow us to match and
		// update the exact chunk we need to update
		output.selector = maxBeaconMatchLowerQuery
	}

	return output
}
