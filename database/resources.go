package database

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"

	mgo "gopkg.in/mgo.v2"

	"github.com/ocmdev/mgorus"
	"github.com/ocmdev/mgosec"
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
	session, err := connectToMongoDB(&conf.MongoDBConfig, log)
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

	//bundle up the system resources
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

	//Begin logging to the metadatabase
	if conf.LogConfig.LogToDB {
		log.Hooks.Add(
			mgorus.NewHookerFromSession(
				session, conf.BroConfig.MetaDB, conf.LogConfig.RitaLogTable,
			),
		)
	}
	return r
}

//connectToMongoDB connects to MongoDB possibly with authentication and TLS
func connectToMongoDB(conf *config.MongoDBCfg, logger *log.Logger) (*mgo.Session, error) {
	if conf.TLS.Enabled {
		authMechanism, err := mgosec.ParseMongoAuthMechanism(conf.AuthMechanism)
		if err != nil {
			authMechanism = mgosec.None
			logger.WithFields(log.Fields{
				"authMechanism": conf.AuthMechanism,
			}).Error(err.Error())
			fmt.Println("[!] Could not parse MongoDB authentication mechanism")
		}

		tlsConf := &tls.Config{}
		if len(conf.TLS.CAFile) > 0 {
			pem, err := ioutil.ReadFile(conf.TLS.CAFile)
			if err != nil {
				logger.WithFields(log.Fields{
					"CAFile": conf.TLS.CAFile,
				}).Error(err.Error())
				fmt.Println("[!] Could not read MongoDB CA file")
			} else {
				tlsConf.RootCAs = x509.NewCertPool()
				tlsConf.RootCAs.AppendCertsFromPEM(pem)
			}
		}
		return mgosec.Dial(conf.ConnectionString, authMechanism, tlsConf)
	}
	return mgo.Dial(conf.ConnectionString)
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
	}))
}
