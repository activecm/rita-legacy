// +build integration

package host

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

var testHost = map[string]*Input{
	"test": &Input{
		Host: data.UniqueIP{
			IP:          "1.2.3.4",
			NetworkUUID: util.PublicNetworkUUID,
			NetworkName: util.PublicNetworkName,
		},
		ConnectionCount: 12,
		TotalBytes:      123,
		TotalDuration:   123.0,
		MaxDuration:     12,
	},
}

func TestUpsert(t *testing.T) {
	testRepo.Upsert(testHost)
}

// TestMain wraps all tests with the needed initialized mock DB and fixtures
func TestMain(m *testing.M) {
	// Store temporary databases files in a temporary directory
	tempDir, _ := ioutil.TempDir("", "testing")
	Server.SetPath(tempDir)

	// Set the main session variable to the temporary MongoDB instance
	res := resources.InitTestResources()

	testRepo = NewMongoRepository(res.DB, res.Config, res.Log)

	// Run the test suite
	retCode := m.Run()

	// Shut down the temporary server and removes data on disk.
	Server.Stop()

	// call with result of m.Run()
	os.Exit(retCode)
}
