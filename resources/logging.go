package resources

import (
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/globalsign/mgo"
	log "github.com/sirupsen/logrus"

	"github.com/activecm/mgorus"
	"github.com/activecm/rita/config"
	"github.com/activecm/rita/util"
	"github.com/rifflock/lfshook"
)

// initLogger creates the logger for logging to stdout
func initLogger(logConfig *config.LogStaticCfg) *log.Logger {
	var logs = &log.Logger{}

	logs.Formatter = new(log.TextFormatter)

	logs.Out = ioutil.Discard
	logs.Hooks = make(log.LevelHooks)

	switch logConfig.LogLevel {
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
	return logs
}

func addFileLogger(logger *log.Logger, logPath string) error {
	time := time.Now().Format(util.TimeFormat)
	logPath = path.Join(logPath, time)
	_, err := os.Stat(logPath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(logPath, 0755)
		if err != nil {
			return err
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
	return nil
}

func addMongoLogger(logger *log.Logger, ssn *mgo.Session, database string, collection string) error {
	err := ssn.DB(database).C(collection).Create(&mgo.CollectionInfo{})

	if err != nil {
		switch err.(type) {
		case *mgo.QueryError:
			//check if create failed because collection already exists
			//https://github.com/mongodb/mongo/blob/master/src/mongo/base/error_codes.err
			queryErr := err.(*mgo.QueryError)
			if queryErr.Code != 48 {
				return err
			}
		default:
			return err
		}
	}
	logger.Hooks.Add(
		mgorus.NewHookerFromSession(
			ssn, database, collection,
		),
	)
	return nil
}
