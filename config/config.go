package config

import (
	"os"
	"reflect"

	"github.com/creasty/defaults"
)

// Version is filled at compile time with the git version of RITA
// Version is filled by "git describe --abbrev=0 --tags"
var Version = "undefined"

// ExactVersion is filled at compile time with the git version of RITA
// ExactVersion is filled by "git describe --always --long --dirty --tags"
var ExactVersion = "undefined"

type (
	//Config holds the configuration for the running system
	Config struct {
		R RunningCfg
		S StaticCfg
		T TableCfg
	}
)

// defaultConfigPath specifies the path of RITA's static config file
const defaultConfigPath = "/etc/rita/config.yaml"

// LoadConfig initializes a Config struct with values read
// from a config file. It takes a string for the path to the file.
// If the string is empty it uses the default path.
func LoadConfig(customConfigPath string) (*Config, error) {
	// Use the default path unless a custom path is given
	configPath := defaultConfigPath
	if customConfigPath != "" {
		configPath = customConfigPath
	}

	config := &Config{}

	// Initialize table config to the default values
	if err := defaults.Set(&config.T); err != nil {
		return nil, err
	}

	// Initialize static config to the default values
	if err := defaults.Set(&config.S); err != nil {
		return nil, err
	}

	// Read the contents from the config file
	contents, err := readStaticConfigFile(configPath)
	if err != nil {
		return nil, err
	}

	// Deserialize the yaml file contents into the static config
	if err := parseStaticConfig(contents, &config.S); err != nil {
		return nil, err
	}

	// Use the static config to initialize the running config
	if err := initRunningConfig(&config.S, &config.R); err != nil {
		return nil, err
	}

	return config, nil
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
