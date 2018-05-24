package beacon

type collectorTestData struct {
	src         string
	dst         string
	ts          []int64
	ds          []int64
	description string
}

var collectorTestDataHostList = []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}

var collectorTestDataList = []collectorTestData{
	collectorTestData{
		src:         collectorTestDataHostList[0],
		dst:         collectorTestDataHostList[1],
		ts:          analyzerTestDataList[0].ts,
		ds:          analyzerTestDataList[0].ds,
		description: "Perfect 24hr Beacon from " + collectorTestDataHostList[0] + " to " + collectorTestDataHostList[1],
	},
	collectorTestData{
		src:         collectorTestDataHostList[1],
		dst:         collectorTestDataHostList[0],
		ts:          analyzerTestDataList[0].ts,
		ds:          analyzerTestDataList[0].ds,
		description: "Perfect 24hr Beacon from " + collectorTestDataHostList[1] + " to " + collectorTestDataHostList[0],
	},
	collectorTestData{
		src:         collectorTestDataHostList[0],
		dst:         collectorTestDataHostList[2],
		ts:          analyzerTestDataList[3].ts,
		ds:          analyzerTestDataList[3].ds,
		description: "Perfect 1hr Beacon from " + collectorTestDataHostList[0] + " to " + collectorTestDataHostList[2],
	},
}
