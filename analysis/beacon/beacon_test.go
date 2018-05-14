package beacon

import (
	"testing"

	"github.com/activecm/rita/database"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestAnalysis(t *testing.T) {
	res := database.InitMockResources("")
	res.Log.Level = log.DebugLevel
	res.Config.S.Beacon.DefaultConnectionThresh = 2

	beaconing := newBeacon(res)
	//set the wait time for the beaconing analysis
	beaconing.analysisWg.Add(1)
	//Open a channel to the analyze function
	go beaconing.analyze()

	//Now we want to iterate through all test cases in beacon_test_data.go
	for _, val := range testDataList {
		//set first and last connection times
		beaconing.minTime = val.ts[0]
		beaconing.maxTime = val.ts[len(val.ts)-1]

		//Now fill in the data that we will need to analyze traffic
		data := &beaconAnalysisInput{
			src:           "0.0.0.0",
			dst:           "0.0.0.0",
			ts:            val.ts, //these are the timestamps
			orig_ip_bytes: val.ds, //these are the data sizes
		}

		//Feed the data into our new channel
		beaconing.analysisChannel <- data

		//Now set our result to the output to our writeChannel
		res := <-beaconing.writeChannel

		t.Run(val.description, func(t *testing.T) {
			if res.Score < val.minScore || res.Score > val.maxScore {
				t.Logf("Expected Score: %f < x < %f\n Score: %f", val.minScore, val.maxScore, res.Score)
				t.Fail()
			}
		})
	}

	//Clonse this input channel
	close(beaconing.analysisChannel)
	//now we wait for the time we specified earlier
	beaconing.analysisWg.Wait()
}

func TestCreateCountMap(t *testing.T) {
	testData := []int64{3, 4, -1, -4, -3, -1, 0, 0, 0, 0, 0, 1, 2, 3, 4, 2, 3, 4, 4}
	testDataCounts := map[int64]int64{
		-4: 1,
		-3: 1,
		-1: 2,
		0:  5,
		1:  1,
		2:  2,
		3:  3,
		4:  4,
	}
	//grab the keys from testDataCounts
	uniqTestData := make([]int64, len(testDataCounts))
	i := 0
	for k := range testDataCounts {
		uniqTestData[i] = k
		i++
	}

	uniq, uniqCounts, mode, modeCount := createCountMap(testData)
	assert.ElementsMatch(t, uniq, uniqTestData)
	for i, datum := range uniq {
		assert.Equal(t, testDataCounts[datum], uniqCounts[i])
	}
	assert.Equal(t, int64(0), mode)
	assert.Equal(t, int64(5), modeCount)
}
