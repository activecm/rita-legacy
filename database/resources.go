package database

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"

	mgo "gopkg.in/mgo.v2"

	"github.com/activecm/mgorus"
	"github.com/activecm/mgosec"
	"github.com/activecm/rita/config"
	"github.com/activecm/rita/util"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
)

type (
	// Resources provides a data structure for passing system Resources
	Resources struct {
		Config *config.Config
		Log    *log.Logger
		DB     *DB
		MetaDB *MetaDBHandle
	}
)

// InitResources grabs the configuration file and intitializes the configuration data
// returning a *Resources object which has all of the necessary configuration information
func InitResources(userConfig string) *Resources {
	//GetConfig requires a table config. "" tells the configuration manager
	//to use the default table config.
	conf, err := config.GetConfig(userConfig, "")
	if err != nil {
		fmt.Fprintf(os.Stdout, "Failed to config, exiting")
		panic(err)
	}

	// Fire up the logging system
	log, err := initLog(conf.S.Log.LogLevel)
	if err != nil {
		fmt.Printf("Failed to prep logger: %s", err.Error())
		os.Exit(-1)
	}
	if conf.S.Log.LogToFile {
		addFileLogger(log, conf.S.Log.RitaLogPath)
	}

	// Jump into the requested database
	session, err := connectToMongoDB(&conf.S.MongoDB, &conf.R.MongoDB, log)
	if err != nil {
		fmt.Printf("Failed to connect to database: %s", err.Error())
		os.Exit(-1)
	}
	session.SetSocketTimeout(conf.S.MongoDB.SocketTimeout)
	session.SetSyncTimeout(conf.S.MongoDB.SocketTimeout)
	session.SetCursorTimeout(0)

	// Allows code to interact with the database
	db := &DB{
		Session: session,
	}

	// Allows code to create and remove tracked databases
	metaDB := &MetaDBHandle{
		DB:   conf.S.Bro.MetaDB,
		lock: new(sync.Mutex),
	}

	//bundle up the system resources
	r := &Resources{
		Log:    log,
		Config: conf,
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

	//Begin logging to the metadatabase
	if conf.S.Log.LogToDB {
		log.Hooks.Add(
			mgorus.NewHookerFromSession(
				session, conf.S.Bro.MetaDB, conf.T.Log.RitaLogTable,
			),
		)
	}
	return r
}

//connectToMongoDB connects to MongoDB possibly with authentication and TLS
func connectToMongoDB(static *config.MongoDBStaticCfg,
	running *config.MongoDBRunningCfg,
	logger *log.Logger) (*mgo.Session, error) {
	if static.TLS.Enabled {
		return mgosec.Dial(static.ConnectionString, running.AuthMechanismParsed, running.TLS.TLSConfig)
	}
	return mgosec.DialInsecure(static.ConnectionString, running.AuthMechanismParsed)
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
	time := time.Now().Format(util.TimeFormat)
	logPath = path.Join(logPath, time)
	_, err := os.Stat(logPath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(logPath, 0755)
		if err != nil {
			fmt.Println("[!] Could not initialize file logger. Check RitaLogDir.")
			return
		}
	}

	logger.Hooks.Add(lfshook.NewHook(lfshook.PathMap{
		log.DebugLevel: path.Join(logPath, "debug.log"),
		log.InfoLevel:  path.Join(logPath, "info.log"),
		log.WarnLevel:  path.Join(logPath, "warn.log"),
		log.ErrorLevel: path.Join(logPath, "error.log"),
		log.FatalLevel: path.Join(logPath, "fatal.log"),
		log.PanicLevel: path.Join(logPath, "panic.log"),
	}, nil))
}
