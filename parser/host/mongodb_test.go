package host

import (
	"io/ioutil"
	"os"
	"testing"

	"gopkg.in/mgo.v2/dbtest"
)

// Server holds the dbtest DBServer
var Server dbtest.DBServer

// Set the test database
var testTargetDB = "tmp_test_db"

var testRepo Repository

func TestCreateIndexes(t *testing.T) {
	err := testRepo.CreateIndexes(testTargetDB)
	if err != nil {
		t.Errorf("Error creating host indexes")
	}
}

// TestMain wraps all tests with the needed initialized mock DB and fixtures
func TestMain(m *testing.M) {
	// Store temporary databases files in a temporary directory
	tempDir, _ := ioutil.TempDir("", "testing")
	Server.SetPath(tempDir)

	// Set the main session variable to the temporary MongoDB instance
	Session := Server.Session()

	mPool := mgosession.NewPool(nil, Session, 1)
	defer mPool.Close()

	testRepo = NewMongoRepository(mPool)

	// Run the test suite
	retCode := m.Run()

	// Clean up test database and session
	Session.DB(testTargetDB).DropDatabase()
	Session.Close()

	// Shut down the temporary server and removes data on disk.
	Server.Stop()

	// call with result of m.Run()
	os.Exit(retCode)
}