package resources

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/activecm/rita-legacy/config"
	"github.com/rifflock/lfshook"
)

// DayFormat stores a correctly formatted timestamp for the day
const DayFormat string = "2006-01-02"

// initLogger creates the logger for logging to stdout and file
func initLogger(logConfig *config.LogStaticCfg) *log.Logger {
	var logs = &log.Logger{}

	logs.Formatter = new(log.TextFormatter)

	logs.Out = ioutil.Discard
	logs.Hooks = make(log.LevelHooks)

	switch logConfig.LogLevel {
	case 3:
		logs.Level = log.DebugLevel
	case 2:
		logs.Level = log.InfoLevel
	case 1:
		logs.Level = log.WarnLevel
	case 0:
		logs.Level = log.ErrorLevel
	}
	if logConfig.LogToFile {
		addFileLogger(logs, logConfig.RitaLogPath)
	}
	return logs
}

func addFileLogger(logger *log.Logger, logPath string) {
	_, err := os.Stat(logPath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(logPath, 0755)
		if err != nil {
			fmt.Println("[!] Could not initialize file logger. Check RitaLogDir.")
			return
		}
	}

	time := time.Now().Format(DayFormat)
	logFile := time + ".log"
	logger.Hooks.Add(lfshook.NewHook(lfshook.PathMap{
		log.DebugLevel: path.Join(logPath, logFile),
		log.InfoLevel:  path.Join(logPath, logFile),
		log.WarnLevel:  path.Join(logPath, logFile),
		log.ErrorLevel: path.Join(logPath, logFile),
		log.FatalLevel: path.Join(logPath, logFile),
		log.PanicLevel: path.Join(logPath, logFile),
	}, nil))
}
