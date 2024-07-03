package resources

import (
	"fmt"
	"os"

	"github.com/activecm/mgorus"
	"github.com/activecm/rita-legacy/config"
	"github.com/activecm/rita-legacy/database"
	log "github.com/sirupsen/logrus"
)

type (
	// Resources provides a data structure for passing system Resources
	Resources struct {
		Config *config.Config
		Log    *log.Logger
		DB     *database.DB
		MetaDB *database.MetaDB
	}
)

// InitResources grabs the configuration file and intitializes the configuration data
// returning a *Resources object which has all of the necessary configuration information
func InitResources(userConfig string) *Resources {
	conf, err := config.LoadConfig(userConfig)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Failed to config: %s\n", err.Error())
		os.Exit(-1)
	}

	// Fire up the logging system
	log := initLogger(&conf.S.Log)

	// Allows code to interact with the database
	db, err := database.NewDB(conf, log)
	if err != nil {
		fmt.Printf("Failed to connect to database: %s\n", err.Error())
		os.Exit(-1)
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
