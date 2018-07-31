package beacon

type collectorTestData struct {
	src         string
	dst         string
	ts          []int64
	ds          []int64
	proto       string
	origPackets int64
	respPackets int64
	description string
}

var collectorTestDataHostList = []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}

var collectorTestDataList = []collectorTestData{
	collectorTestData{
		src:   collectorTestDataHostList[0],
		dst:   collectorTestDataHostList[1],
		ts:    analyzerTestDataList[0].ts,
		ds:    analyzerTestDataList[0].ds,
		proto: "udp",
		//resp and orig packets are zero to ensure udp isn't affected by the tcp threshold
		description: "Perfect 24hr Beacon from " + collectorTestDataHostList[0] + " to " + collectorTestDataHostList[1],
	},
	collectorTestData{
		src:         collectorTestDataHostList[0],
		dst:         collectorTestDataHostList[2],
		ts:          analyzerTestDataList[3].ts,
		ds:          analyzerTestDataList[3].ds,
		proto:       "udp",
		description: "Perfect 1hr Beacon from " + collectorTestDataHostList[0] + " to " + collectorTestDataHostList[2],
	},
	collectorTestData{
		src:         collectorTestDataHostList[1],
		dst:         collectorTestDataHostList[0],
		ts:          analyzerTestDataList[0].ts,
		ds:          analyzerTestDataList[0].ds,
		proto:       "udp",
		description: "Perfect 24hr Beacon from " + collectorTestDataHostList[1] + " to " + collectorTestDataHostList[0],
	},
	collectorTestData{
		src:         collectorTestDataHostList[1],
		dst:         collectorTestDataHostList[2],
		ts:          []int64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200, 210, 220, 230, 240, 250},
		ds:          []int64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200, 210, 220, 230, 240, 250},
		proto:       "tcp",
		origPackets: 1,
		respPackets: 1,
		description: "Invalid TCP Beacon With orig + resp IP packets = 2 from " + collectorTestDataHostList[1] + " to " + collectorTestDataHostList[2],
	},
	collectorTestData{
		src:         collectorTestDataHostList[2],
		dst:         collectorTestDataHostList[0],
		ts:          []int64{1, 2, 3, 4, 5},
		ds:          []int64{1, 1, 1, 1, 1},
		proto:       "udp",
		description: "Invalid short beacon from " + collectorTestDataHostList[2] + " to " + collectorTestDataHostList[0],
	},
	collectorTestData{
		src:         collectorTestDataHostList[2],
		dst:         collectorTestDataHostList[1],
		ts:          []int64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200, 210, 220, 230, 240, 250},
		ds:          []int64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200, 210, 220, 230, 240, 250},
		proto:       "tcp",
		origPackets: 5,
		respPackets: 5,
		description: "Valid TCP Beacon With orig + resp IP packets = 10 from " + collectorTestDataHostList[2] + " to " + collectorTestDataHostList[1],
	},
}
