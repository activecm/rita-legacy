package beacon

import (
	"testing"

	"github.com/activecm/rita/datatypes/structure"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/resources"
	"github.com/stretchr/testify/assert"
)

func TestCollector(t *testing.T) {
	res := resources.InitIntegrationTestingResources(t)
	setUpCollectorTest(t, res)
	defer tearDownCollectorConnRecords(t, res)

	collectedChannel := make(chan *beaconAnalysisInput)

	collector := newCollector(
		res.DB, res.Config, res.Config.S.Beacon.DefaultConnectionThresh,
		func(collected *beaconAnalysisInput) {
			collectedChannel <- collected
		},
	)
	collector.start()
	collector.collect(collectorTestDataHostList[0])

	t.Run(collectorTestDataList[0].description, func(t *testing.T) {
		collectedHost := <-collectedChannel
		collectionSuccessful(t, &collectorTestDataList[0], collectedHost)
	})

	t.Run(collectorTestDataList[2].description, func(t *testing.T) {
		collectedHost := <-collectedChannel
		collectionSuccessful(t, &collectorTestDataList[2], collectedHost)
	})

	collector.collect(collectorTestDataHostList[1])

	t.Run(collectorTestDataList[1].description, func(t *testing.T) {
		collectedHost := <-collectedChannel
		collectionSuccessful(t, &collectorTestDataList[1], collectedHost)
	})

	collector.flush()
}

func collectionSuccessful(t *testing.T, collectorTestData *collectorTestData, collectedData *beaconAnalysisInput) {
	assert.Equal(t, collectorTestData.src, collectedData.src)
	assert.Equal(t, collectorTestData.dst, collectedData.dst)
	for i := range collectorTestData.ts {
		assert.Equal(t, collectorTestData.ts[i], collectedData.ts[i])
		assert.Equal(t, collectorTestData.ds[i], collectedData.origIPBytes[i])
	}
}

func setUpCollectorTest(t *testing.T, res *resources.Resources) {
	session := res.DB.Session.Copy()
	defer session.Close()
	for _, record := range collectorTestDataList {
		if len(record.ts) != len(record.ds) {
			t.FailNow()
		}
		uconn := structure.UniqueConnection{
			Src:             record.src,
			Dst:             record.dst,
			ConnectionCount: len(record.ts),
		}
		session.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.UniqueConnTable).Insert(&uconn)
		for i, timestamp := range record.ts {
			dataSize := record.ds[i]
			connRecord := parsetypes.Conn{
				TimeStamp:   timestamp,
				OrigIPBytes: dataSize,
				Source:      record.src,
				Destination: record.dst,
			}
			session.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.ConnTable).Insert(&connRecord)
		}
	}
}

func tearDownCollectorConnRecords(t *testing.T, res *resources.Resources) {
	session := res.DB.Session.Copy()
	defer session.Close()
	session.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.ConnTable).DropCollection()
	session.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.UniqueConnTable).DropCollection()
}
