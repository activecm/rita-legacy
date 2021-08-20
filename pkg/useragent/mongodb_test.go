// +build integration

package useragent

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

var testUserAgent = map[string]*Input{
	"Debian APT-HTTP/1.3 (1.2.24)": {
		Seen: 123,
	},
}

func init() {
	testUserAgent["Debian APT-HTTP/1.3 (1.2.24)"].OrigIps.Insert(
		data.UniqueIP{
			IP:          "5.6.7.8",
			NetworkUUID: util.PublicNetworkUUID,
			NetworkName: util.PublicNetworkName,
		},
	)
	testUserAgent["Debian APT-HTTP/1.3 (1.2.24)"].OrigIps.Insert(
		data.UniqueIP{
			IP:          "9.10.11.12",
			NetworkUUID: util.PublicNetworkUUID,
			NetworkName: util.PublicNetworkName,
		},
	)
}

func TestUpsert(t *testing.T) {
	testRepo.Upsert(testUserAgent)

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
