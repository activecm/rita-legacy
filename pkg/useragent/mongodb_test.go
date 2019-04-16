package useragent

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

var testUserAgent = map[string]*Input{
	"Debian APT-HTTP/1.3 (1.2.24)": &Input{
		OrigIps:  []string{"1.2.3.4", "1.1.1.1"},
		Seen: 123,
	},
}

func TestCreateIndexes(t *testing.T) {
	err := testRepo.CreateIndexes()
	if err != nil {
		t.Errorf("Error creating useragent indexes")
	}
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

	testRepo = NewMongoRepository(res)

	// Run the test suite
	retCode := m.Run()

	// Shut down the temporary server and removes data on disk.
	Server.Stop()

	// call with result of m.Run()
	os.Exit(retCode)
}
