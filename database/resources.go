package database

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	mgo "gopkg.in/mgo.v2"

	"github.com/Zalgo2462/mgorus"
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/util"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
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
	log, err := initLog(conf.LogConfig.LogLevel)
	if err != nil {
		fmt.Printf("Failed to prep logger: %s", err.Error())
		os.Exit(-1)
	}
	if conf.LogConfig.LogToFile {
		addFileLogger(log, conf.LogConfig.RitaLogPath)
	}

	// Jump into the requested database
	session, err := mgo.Dial(conf.DatabaseHost)
	if err != nil {
		fmt.Printf("Failed to connect to database: %s", err.Error())
		os.Exit(-1)
	}
	session.SetSocketTimeout(2 * time.Hour)
	session.SetSyncTimeout(2 * time.Hour)
	session.SetCursorTimeout(0)

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
		metaDB.createMetaDB()
	}
	if conf.LogConfig.LogToDB {
		addMongoLogger(log, conf.DatabaseHost, conf.BroConfig.MetaDB,
			conf.LogConfig.RitaLogTable)
	}
	return r
}

// initLog creates the logger for logging to stdout and file
func initLog(level int) (*log.Logger, error) {
	var logs = &log.Logger{}

	logs.Formatter = new(log.TextFormatter)

	logs.Out = ioutil.Discard
	logs.Hooks = make(log.LevelHooks)

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

func addFileLogger(logger *log.Logger, logPath string) {
	logger.Hooks.Add(lfshook.NewHook(lfshook.PathMap{
		log.DebugLevel: logPath + "/debug-" + time.Now().Format(util.TimeFormat) + ".log",
		log.InfoLevel:  logPath + "/info-" + time.Now().Format(util.TimeFormat) + ".log",
		log.WarnLevel:  logPath + "/warn-" + time.Now().Format(util.TimeFormat) + ".log",
		log.ErrorLevel: logPath + "/error-" + time.Now().Format(util.TimeFormat) + ".log",
	}))
}

func addMongoLogger(logger *log.Logger, dbHost, metaDB, logColl string) error {
	mgoHook, err := mgorus.NewHooker(dbHost, metaDB, logColl)
	if err == nil {
		logger.Hooks.Add(mgoHook)
	}
	return err
}
