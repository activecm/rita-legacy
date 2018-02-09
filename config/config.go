package config

import (
	"os"
	"reflect"
)

//Version is filled at compile time with the git version of RITA
//Version is filled by "git describe --abbrev=0 --tags"
var Version = "undefined"

//ExactVersion is filled at compile time with the git version of RITA
//ExactVersion is filled by "git describe --always --long --dirty --tags"
var ExactVersion = "undefined"

type (
	//Config holds the configuration for the running system
	Config struct {
		R RunningCfg
		S StaticCfg
		T TableCfg
	}
)

//userConfigPath specifies the path of RITA's static config file
//relative to the user's home directory
const userConfigPath = "/etc/rita/config.yaml"

//tableConfigPath specifies teh path of RITA's table config file
//relative to the user's home directory
const tableConfigPath = "/etc/rita/tables.yaml"

//NOTE: If go ever gets default parameters, default the config options to ""

// GetConfig retrieves a configuration in order of precedence
func GetConfig(userConfig string, tableConfig string) (*Config, error) {
	if userConfig == "" {
		userConfig = userConfigPath
	}
	if tableConfig == "" {
		tableConfig = tableConfigPath
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
