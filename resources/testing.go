package resources

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
)

//InitIntegrationTestingResources creates a default testing
//resource bundle for use with integration testing.
//The MongoDB server is contacted via the URI provided
//as by go test -args [MongoDB URI].
func InitIntegrationTestingResources(t *testing.T) *Resources {
	if testing.Short() {
		t.Skip()
	}

	mongoURI := os.Args[len(os.Args)-1]

	if !strings.Contains(mongoURI, "mongodb://") {
		t.Fatal("-args [MongoDB URI] is required to run RITA integration tests with go test")
	}

	conf, err := config.LoadTestingConfig(mongoURI)
	if err != nil {
		t.Fatal(err)
	}

	// Fire up the logging system
	log := initLogger(&conf.S.Log)

	// Allows code to interact with the database
	db, err := database.NewDB(conf, log)
	if err != nil {
		t.Fatal(err)
	}

	// Allows code to keep track of database metadata
	databaseIndex, err := database.NewRITADatabaseIndex(
		db.Session, conf.S.Bro.MetaDB, conf.T.Meta.DatabasesTable, log,
	)

	if err != nil {
		fmt.Printf("Failed to load RITA database index: %s\n", err.Error())
		t.Fatal(err)
	}

	// Allows code to keep track of which files have already been imported
	fileIndex, err := database.NewImportedFilesIndex(
		db.Session, conf.S.Bro.MetaDB, conf.T.Meta.FilesTable, log,
	)

	if err != nil {
		fmt.Printf("Failed to load the index of imported files: %s\n", err.Error())
		t.Fatal(err)
	}

	//bundle up the system resources
	r := &Resources{
		Config:    conf,
		Log:       log,
		DB:        db,
		DBIndex:   databaseIndex,
		FileIndex: fileIndex,
	}
	return r
}
