package config

import (
	"fmt"
	"os"
	"os/user"
	"reflect"
)

//VERSION is filled at compile time with the git version of RITA
var VERSION = "undefined"

type (
	//Config holds the configuration for the running system
	Config struct {
		R RunningCfg
		S StaticCfg
		T TableCfg
	}
)

const userConfigPath = "/.rita/config.yaml"
const tableConfigPath = "/.rita/tables.yaml"

//NOTE: If go ever gets default parameters, default the config options to ""

// GetConfig retrieves a configuration in order of precedence
func GetConfig(userConfig string, tableConfig string) (*Config, error) {
	//var user string
	var currUser *user.User
	if userConfig == "" || tableConfig == "" {
		// Get the user's homedir
		var err error
		currUser, err = user.Current()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not get user info: %s\n", err.Error())
			return nil, err
		}
	}

	if userConfig == "" {
		userConfig = currUser.HomeDir + userConfigPath
	}
	if tableConfig == "" {
		tableConfig = currUser.HomeDir + tableConfigPath
	}

	return loadSystemConfig(userConfig, tableConfig)
}

// loadSystemConfig attempts to parse a config file
func loadSystemConfig(userConfig string, tableConfig string) (*Config, error) {
	var config = new(Config)
	static, err := loadStaticConfig(userConfig)
	if err != nil {
		return config, err
	}
	config.S = *static

	tables, err := loadTableConfig(tableConfig)
	if err != nil {
		return config, err
	}
	config.T = *tables

	running, err := loadRunningConfig(static)
	if err != nil {
		return config, err
	}
	config.R = *running

	return config, err
}

// expandConfig expands environment variables in config strings
func expandConfig(reflected reflect.Value) {
	for i := 0; i < reflected.NumField(); i++ {
		f := reflected.Field(i)
		// process sub configs
		if f.Kind() == reflect.Struct {
			expandConfig(f)
		} else if f.Kind() == reflect.String {
			f.SetString(os.ExpandEnv(f.String()))
		} else if f.Kind() == reflect.Slice && f.Type().Elem().Kind() == reflect.String {
			strs := f.Interface().([]string)
			for i, str := range strs {
				strs[i] = os.ExpandEnv(str)
			}
			f.Set(reflect.ValueOf(strs))
		}
	}
}
