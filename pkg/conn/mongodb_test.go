package conn

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/dbtest"
)

// Server holds the dbtest DBServer
var Server dbtest.DBServer

// Set the test database
var testTargetDB = "tmp_test_db"

var testRepo Repository

func TestBulkDelete(t *testing.T) {
	testConns := []*parsetypes.Conn{
		{Source: "127.0.0.1", Destination: "127.0.0.1"},
		{Source: "127.0.0.1", Destination: "127.0.0.1"},
	}

	err := testRepo.BulkDelete(testConns)
	if err != nil {
		t.Errorf("Error bulk deleting conns")
	}
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
