package database

import (
	"fmt"
	"os"
	"sync"

	mgo "gopkg.in/mgo.v2"

	log "github.com/Sirupsen/logrus"
	"github.com/ocmdev/rita/config"
)

type (
	// Resources provides a data structure for passing system Resources
	Resources struct {
		System *config.SystemConfig
		Log    *log.Logger
		DB     *DB
		MetaDB *MetaDBHandle
	}
)

// InitResources grabs the configuration file and intitializes the configuration data
// returning a *Resources object which has all of the necessary configuration information
func InitResources(cfgPath string) *Resources {
	conf, ok := config.GetConfig(cfgPath)
	if !ok {
		fmt.Fprintf(os.Stdout, "Failed to config, exiting")
		os.Exit(-1)
	}

	// Fire up the logging system
	log, err := initLog(conf.LogLevel, conf.LogType)
	if err != nil {
		fmt.Printf("Failed to prep logger: %s", err.Error())
		os.Exit(-1)
	}

	// Jump into the requested database
	session, err := mgo.Dial(conf.DatabaseHost)
	if err != nil {
		fmt.Printf("Failed to connect to database: %s", err.Error())
		os.Exit(-1)
	}

	// Allows code to interact with the database
	db := &DB{
		Session: session,
	}

	// Allows code to create and remove tracked databases
	metaDB := &MetaDBHandle{
		DB:   conf.BroConfig.MetaDB,
		lock: new(sync.Mutex),
	}

	r := &Resources{
		Log:    log,
		System: conf,
	}

	// db and resources have cyclic pointers
	r.DB = db
	db.resources = r

	// metadb and resources have cyclic pointers
	r.MetaDB = metaDB
	metaDB.res = r

	//Build Meta collection
	if !metaDB.isBuilt() {
		metaDB.newMetaDBHandle()
	}

	return r
}

/*
 * Name:     InitLog
 * Purpose:  Initializes logging utilities
 * comments:
 */
func initLog(level int, logtype string) (*log.Logger, error) {
	var logs = &log.Logger{}

	if logtype == "production" {
		logs.Formatter = new(log.JSONFormatter)
	} else {
		logs.Formatter = new(log.TextFormatter)
	}

	logs.Out = os.Stderr

	switch level {
	case 3:
		logs.Level = log.DebugLevel
		break
	case 2:
		logs.Level = log.InfoLevel
		break
	case 1:
		logs.Level = log.WarnLevel
		break
	case 0:
		logs.Level = log.ErrorLevel
	}

	return logs, nil
}
