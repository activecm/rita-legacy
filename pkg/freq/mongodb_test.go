package freq

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/activecm/rita/database"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/globalsign/mgo/dbtest"
)

// Server holds the dbtest DBServer
var Server dbtest.DBServer

// Set the test database
var testTargetDB = "tmp_test_db"

var testRepo Repository

func TestInsert(t *testing.T) {
	testFreq := &parsetypes.Freq{
		Source:          "127.0.0.1",
		Destination:     "127.0.0.1",
		ConnectionCount: 12,
	}

	err := testRepo.Insert(testFreq, testTargetDB)
	if err != nil {
		t.Errorf("Error inserting to freq collection")
	}
}

// TestMain wraps all tests with the needed initialized mock DB and fixtures
func TestMain(m *testing.M) {
	// Store temporary databases files in a temporary directory
	tempDir, _ := ioutil.TempDir("", "testing")
	Server.SetPath(tempDir)

	// Set the main session variable to the temporary MongoDB instance
	ssn := Server.Session()

	db := database.DB{Session: ssn}

	testRepo = NewMongoRepository(&db)

	// Run the test suite
	retCode := m.Run()

	// Clean up test database and session
	ssn.DB(testTargetDB).DropDatabase()
	ssn.Close()

	// Shut down the temporary server and removes data on disk.
	Server.Stop()

	// call with result of m.Run()
	os.Exit(retCode)
}
