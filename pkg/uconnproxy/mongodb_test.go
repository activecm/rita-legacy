// +build integration

package uconnproxy

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo/dbtest"
)

// Server holds the dbtest DBServer
var Server dbtest.DBServer

// Set the test database
var testTargetDB = "tmp_test_db"

var testRepo Repository

var testUconn = map[string]*Input{
	"test": &Input{
		Hosts: data.UniqueIPPair{
			SrcIP:          "127.0.0.1",
			SrcNetworkUUID: util.UnknownPrivateNetworkUUID,
			SrcNetworkName: util.UnknownPrivateNetworkName,
			DstIP:          "127.0.0.1",
			DstNetworkUUID: util.UnknownPrivateNetworkUUID,
			DstNetworkName: util.UnknownPrivateNetworkName,
		},
		ConnectionCount: 12,
		IsLocalSrc:      true,
		IsLocalDst:      true,
		TotalBytes:      123,
		TsList:          []int64{1234567, 1234567},
		OrigBytesList:   []int64{12, 12},
		TotalDuration:   123.0,
		MaxDuration:     12,
	},
}

func TestUpsert(t *testing.T) {
	testRepo.Upsert(testUconn)

}

// TestMain wraps all tests with the needed initialized mock DB and fixtures
func TestMain(m *testing.M) {
	// Store temporary databases files in a temporary directory
	tempDir, _ := ioutil.TempDir("", "testing")
	Server.SetPath(tempDir)

	// Set the main session variable to the temporary MongoDB instance
	res := resources.InitTestResources()

	testRepo = NewMongoRepository(res)

	// Run the test suite
	retCode := m.Run()

	// Shut down the temporary server and removes data on disk.
	Server.Stop()

	// call with result of m.Run()
	os.Exit(retCode)
}
