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

//NOTE: If go ever gets default parameters, default the config options to ""

// GetConfig retrieves a configuration in order of precedence
func GetConfig(userConfig string) (*Config, error) {
	var config = new(Config)

	var configSearchPath []string
	if userConfig != "" {
		// Use the user specified path
		configSearchPath = []string { userConfig }
	} else {
		// Search the following paths for a config file and
		// use the first one found
		configSearchPath = []string {
			"./config.yaml",
			"$HOME/.rita/config.yaml",
			"../etc/rita/config.yaml",
			"/etc/rita/config.yaml",
		}
	}
	
	var static *StaticCfg
	var err error
	for _, configPath := range configSearchPath {
		static, err = loadStaticConfig(configPath)
		if err == nil {
			// Stop after finding the first successful file
			config.S = *static
			break
		}
		fmt.Println(err.Error())
	}
	// If none of the config file paths worked, return an error
	if err != nil {
		return config, err
	}

	tables, err := loadTableConfig()
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
