// +build integration

package beacon

import (
	"testing"

	"github.com/activecm/rita/datatypes/beacon"
	"github.com/stretchr/testify/require"

	"github.com/activecm/rita/resources"
)

func TestWriter(t *testing.T) {
	res := resources.InitIntegrationTestingResources(t)
	writer := newWriter(res.DB, res.Config)
	writer.start()
	for i := range writerTestDataList {
		writer.write(&writerTestDataList[i])
	}
	writer.close()

	var writtenData []beacon.AnalysisOutput
	res.DB.Session.DB(res.DB.GetSelectedDB()).C(res.Config.T.Beacon.BeaconTable).Find(nil).All(&writtenData)
	require.ElementsMatch(t, writerTestDataList, writtenData)
}
