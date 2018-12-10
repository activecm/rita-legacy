package beacon

import (
	"sort"
	"testing"

	"github.com/activecm/rita/datatypes/beacon"
	"github.com/activecm/rita/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzer(t *testing.T) {
	for _, val := range analyzerTestDataList {
		analyzedChan := make(chan *beacon.AnalysisOutput, 1)

		analyzer := newAnalyzer(
			val.ts[0], val.ts[len(val.ts)-1], //min max times,
			func(output *beacon.AnalysisOutput) {
				analyzedChan <- output
			}, func() {
				close(analyzedChan)
			},
		)
		analyzer.start()
		analyzer.analyze(&beacon.AnalysisInput{
			Src:         "0.0.0.0",
			Dst:         "0.0.0.0",
			TsList:      val.ts, //these are the timestamps
			OrigIPBytes: val.ds, //these are the data sizes
		})
		analyzer.close()

		t.Run(val.description, func(t *testing.T) {
			output, ok := <-analyzedChan
			require.True(t, ok)
			t.Logf("Expected Score: %f < x < %f\n Score: %f", val.minScore, val.maxScore, output.Score)
			require.False(t, output.Score < val.minScore || output.Score > val.maxScore)
		})
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
	sort.Sort(util.SortableInt64(testData))

	//grab the keys from testDataCounts
	uniqTestData := make([]int64, len(testDataCounts))
	i := 0
	for k := range testDataCounts {
		uniqTestData[i] = k
		i++
	}
	uniq, uniqCounts, mode, modeCount := createCountMap(testData)
	require.ElementsMatch(t, uniq, uniqTestData)
	for i, datum := range uniq {
		require.Equal(t, testDataCounts[datum], uniqCounts[i])
	}
	require.Equal(t, int64(0), mode)
	require.Equal(t, int64(5), modeCount)
}

func TestCountAndRemoveConsecutiveDuplicates(t *testing.T) {
	allSame := []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	allSameExp := []int64{0}
	outOfOrder := []int64{5, 5, 1, 5, 5, 1, 5, 5}
	outOfOrderExp := []int64{5, 1, 5, 1, 5}
	normal := []int64{1, 2, 3, 3, 3, 4, 4, 4, 5, 6, 10}
	normalExp := []int64{1, 2, 3, 4, 5, 6, 10}
	allSameTest, allSameCounts := countAndRemoveConsecutiveDuplicates(allSame)
	assert.Equal(t, allSameExp, allSameTest)
	assert.Equal(t, int64(len(allSame)), allSameCounts[0])

	outOfOrderTest, outOfOrderCounts := countAndRemoveConsecutiveDuplicates(outOfOrder)
	assert.Equal(t, outOfOrderExp, outOfOrderTest)
	assert.Equal(t, int64(6), outOfOrderCounts[5])
	assert.Equal(t, int64(2), outOfOrderCounts[1])

	normalTest, normalCounts := countAndRemoveConsecutiveDuplicates(normal)
	assert.Equal(t, normalExp, normalTest)
	assert.True(t, sort.IsSorted(util.SortableInt64(normalTest)))
	assert.Equal(t, int64(1), normalCounts[1])
	assert.Equal(t, int64(1), normalCounts[2])
	assert.Equal(t, int64(3), normalCounts[3])
	assert.Equal(t, int64(3), normalCounts[4])
	assert.Equal(t, int64(1), normalCounts[5])
	assert.Equal(t, int64(1), normalCounts[6])
	assert.Equal(t, int64(1), normalCounts[10])
}
