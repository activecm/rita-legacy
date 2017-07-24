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

// GetConfig retrieves a configuration in order of precedence
func GetConfig(cfgPath string) (*Config, error) {
	if cfgPath != "" {
		return loadSystemConfig(cfgPath)
	}

	// Get the user's homedir
	user, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get user info: %s\n", err.Error())
	} else {
		return loadSystemConfig(user.HomeDir + "/.rita/config.yaml")
	}

	// If none of the other configs have worked, go for the global config
	return loadSystemConfig("/etc/rita/config.yaml")
}

// loadSystemConfig attempts to parse a config file
func loadSystemConfig(cfgPath string) (*Config, error) {
	var config = new(Config)
	static, err := loadStaticConfig(cfgPath)
	if err != nil {
		return config, err
	}
	config.S = *static

	tables, err := loadTableConfig(cfgPath)
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
