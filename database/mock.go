package database

import (
	"fmt"
	"os"

	"github.com/ocmdev/rita/config"
)

// InitMockResources grabs the configuration file and intitializes the configuration data
// returning a *Resources object which has all of the necessary configuration information
func InitMockResources(cfgPath string) *Resources {
	//TODO: hard code in a test config
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

	// Allows code to interact with the database
	db := &DB{
		//TODO: Mock session
		Session: nil,
	}

	r := &Resources{
		Log:    log,
		System: conf,
	}

	// db and resources have cyclic pointers
	r.DB = db
	db.resources = r
	return r
}
