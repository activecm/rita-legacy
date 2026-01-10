package resources

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/activecm/mgorus"
	"github.com/activecm/rita-legacy/config"
	"github.com/activecm/rita-legacy/database"
)

// InitIntegrationTestingResources creates a default testing
// resource bundle for use with integration testing.
// The MongoDB server is contacted via the URI provided
// as by go test -args [MongoDB URI].
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

	// Allows code to create and remove tracked databases
	metaDB := database.NewMetaDB(conf, db.Session, log)

	//Begin logging to the metadatabase
	if conf.S.Log.LogToDB {
		log.Hooks.Add(
			mgorus.NewHookerFromSession(
				db.Session, conf.S.MongoDB.MetaDB, conf.T.Log.RitaLogTable,
			),
		)
	}

	//bundle up the system resources
	r := &Resources{
		Config: conf,
		Log:    log,
		DB:     db,
		MetaDB: metaDB,
	}
	return r
}

// InitTestResources creates a default testing
// resource bundle for use with integration testing.
func InitTestResources() *Resources {

	conf, err := config.LoadTestingConfig("mongodb://localhost:27017")
	if err != nil {
		fmt.Println(err)
		return nil
	}

	// Fire up the logging system
	log := initLogger(&conf.S.Log)

	// Allows code to interact with the database
	db, err := database.NewDB(conf, log)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	// Allows code to create and remove tracked databases
	metaDB := database.NewMetaDB(conf, db.Session, log)

	//Begin logging to the metadatabase
	if conf.S.Log.LogToDB {
		log.Hooks.Add(
			mgorus.NewHookerFromSession(
				db.Session, conf.S.MongoDB.MetaDB, conf.T.Log.RitaLogTable,
			),
		)
	}

	//bundle up the system resources
	r := &Resources{
		Config: conf,
		Log:    log,
		DB:     db,
		MetaDB: metaDB,
	}
	return r
}
