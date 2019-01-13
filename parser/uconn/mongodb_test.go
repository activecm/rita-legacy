package uconn

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/activecm/rita/parser/parsetypes"
	"gopkg.in/mgo.v2/dbtest"
	"github.com/juju/mgosession"
)

// Server holds the dbtest DBServer
var Server dbtest.DBServer

// Set the test database
var testTargetDB = "tmp_test_db"

var uconnRepo Repository

func TestInsert(t *testing.T) {
	testUconn := &parsetypes.Uconn{
		Source: "127.0.0.1",
		Destination: "127.0.0.1",
		ConnectionCount: 12,
		LocalSource: true,
		LocalDestination: true,
		TotalBytes: 123,
		AverageBytes: 12,
		TSList: []int64{1234567, 1234567},
		OrigBytesList: []int64{12, 12},
		TotalDuration: 123.0,
		MaxDuration: 12,
	}

	err := uconnRepo.Insert(testUconn, testTargetDB)
	if err != nil {
		t.Errorf("Error inserting uconn")
	}
}

// TestMain wraps all tests with the needed initialized mock DB and fixtures
func TestMain(m *testing.M) {
	// The tempdir is created so MongoDB has a location to store its files.
	// Contents are wiped once the server stops
	tempDir, _ := ioutil.TempDir("", "testing")
	Server.SetPath(tempDir)

	// My main session var is now set to the temporary MongoDB instance
	Session := Server.Session()

	mPool := mgosession.NewPool(nil, Session, 1)
	defer mPool.Close()

	uconnRepo = NewMongoRepository(mPool)

	// Make sure to insert my fixtures
	//testing.T.TestInsert()

	// Run the test suite
	retCode := m.Run()

	// Make sure we DropDatabase so we make absolutely sure nothing is left or locked while wiping the data and
	// close session
	Session.DB(targetDB).DropDatabase()
	Session.Close()

	// Stop shuts down the temporary server and removes data on disk.
	Server.Stop()

	// call with result of m.Run()
	os.Exit(retCode)
}