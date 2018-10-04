package resources

import (
	"fmt"
	"os"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	log "github.com/sirupsen/logrus"
)

type (
	// Resources provides a data structure for passing system Resources
	Resources struct {
		Config *config.Config
		Log    *log.Logger
		DB     *database.DB

		DBIndex   database.RITADatabaseIndex
		FileIndex database.ImportedFilesIndex
	}
)

// InitResources grabs the configuration file and intitializes the configuration data
// returning a *Resources object which has all of the necessary configuration information
func InitResources(userConfig string) *Resources {
	conf, err := config.LoadConfig(userConfig)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Failed to load configuration: %s\n", err.Error())
		os.Exit(-1)
	}

	// Fire up the logging system
	log := initLogger(&conf.S.Log)

	if conf.S.Log.LogToFile {
		err = addFileLogger(log, conf.S.Log.RitaLogPath)
		if err != nil {
			fmt.Printf("Could not initialize file logger: %s\n", err.Error())
			os.Exit(-1)
		}
	}

	// Allows code to interact with the database
	db, err := database.NewDB(conf, log)
	if err != nil {
		fmt.Printf("Failed to connect to MongoDB: %s\n", err.Error())
		os.Exit(-1)
	}

	// Allows code to keep track of database metadata
	databaseIndex, err := database.NewRITADatabaseIndex(
		db.Session, conf.S.Bro.MetaDB, conf.T.Meta.DatabasesTable, log,
	)

	if err != nil {
		fmt.Printf("Failed to load RITA database index: %s\n", err.Error())
		os.Exit(-1)
	}

	// Allows code to keep track of which files have already been imported
	fileIndex, err := database.NewImportedFilesIndex(
		db.Session, conf.S.Bro.MetaDB, conf.T.Meta.FilesTable, log,
	)

	if err != nil {
		fmt.Printf("Failed to load the index of imported files: %s\n", err.Error())
		os.Exit(-1)
	}

	//Begin logging to the metadatabase
	if conf.S.Log.LogToDB {
		err = addMongoLogger(log, db.Session, conf.S.Bro.MetaDB, conf.T.Log.RitaLogTable)
		if err != nil {
			fmt.Printf("Could not initialize MongoDB logger: %s\n", err.Error())
			os.Exit(-1)
		}
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
