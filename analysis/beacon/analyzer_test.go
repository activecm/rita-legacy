package beacon

import (
	"testing"

	dataBeacon "github.com/activecm/rita/datatypes/beacon"

	"github.com/stretchr/testify/assert"
)

func TestAnalysis(t *testing.T) {
	for _, val := range testDataList {
		analyzer := newAnalyzer(5, val.ts[0], val.ts[len(val.ts)-1],
			func(res *dataBeacon.BeaconAnalysisOutput) {
				t.Run(val.description, func(t *testing.T) {
					t.Logf("Expected Score: %f < x < %f\n Score: %f", val.minScore, val.maxScore, res.Score)
					if res.Score < val.minScore || res.Score > val.maxScore {
						t.Fail()
					}
				})
			})
		analyzer.start()
		analyzer.analyze(&beaconAnalysisInput{
			src:         "0.0.0.0",
			dst:         "0.0.0.0",
			ts:          val.ts, //these are the timestamps
			origIPBytes: val.ds, //these are the data sizes
		})
		analyzer.flush()
	}
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
