// +build integration

package explodeddns

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/dbtest"
)

// Server holds the dbtest DBServer
var Server dbtest.DBServer

// Set the test database
var testTargetDB = "tmp_test_db"

var testRepo Repository

var testExplodedDNS = map[string]int{
	"a.b.activecountermeasures.com":   123,
	"x.a.b.activecountermeasures.com": 38,
	"activecountermeasures.com":       1,
	"google.com":                      912,
}

func TestUpdateDomains(t *testing.T) {
	testRepo.Upsert(testExplodedDNS)
	// if err != nil {
	// 	t.Errorf("Error creating explodedDNS upserts")
	// }
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
