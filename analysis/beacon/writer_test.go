package beacon

import (
	"testing"

	dataBeacon "github.com/activecm/rita/datatypes/beacon"
	"github.com/stretchr/testify/assert"

	"github.com/activecm/rita/resources"
)

func TestWriter(t *testing.T) {
	res := resources.InitIntegrationTestingResources(t)
	writer := newWriter(res.DB, res.Config)
	writer.start()
	for i := range writerTestDataList {
		writer.write(&writerTestDataList[i])
	}
	writer.flush()

	var writtenData []dataBeacon.BeaconAnalysisOutput
	res.DB.Session.DB(res.DB.GetSelectedDB()).C(res.Config.T.Beacon.BeaconTable).Find(nil).All(&writtenData)
	assert.ElementsMatch(t, writerTestDataList, writtenData)
}
