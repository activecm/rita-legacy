package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"time"

	yaml "gopkg.in/yaml.v2"
)

type (
	//StaticCfg is the container for other static config sections
	StaticCfg struct {
		UserConfig   UserCfgStaticCfg     `yaml:"UserConfig"`
		MongoDB      MongoDBStaticCfg     `yaml:"MongoDB"`
		Log          LogStaticCfg         `yaml:"LogConfig"`
		Blacklisted  BlacklistedStaticCfg `yaml:"BlackListed"`
		Crossref     CrossrefStaticCfg    `yaml:"Crossref"`
		Scanning     ScanningStaticCfg    `yaml:"Scanning"`
		Beacon       BeaconStaticCfg      `yaml:"Beacon"`
		Bro          BroStaticCfg         `yaml:"Bro"`
		Version      string
		ExactVersion string
	}

	//UserCfgStaticCfg contains
	UserCfgStaticCfg struct {
		UpdateCheckFrequency    *int   `yaml:"UpdateCheckFrequency,omitempty"`
	}


	//MongoDBStaticCfg contains the means for connecting to MongoDB
	MongoDBStaticCfg struct {
		ConnectionString string        `yaml:"ConnectionString"`
		AuthMechanism    string        `yaml:"AuthenticationMechanism"`
		SocketTimeout    time.Duration `yaml:"SocketTimeout"`
		TLS              TLSStaticCfg  `yaml:"TLS"`
	}

	//TLSStaticCfg contains the means for connecting to MongoDB over TLS
	TLSStaticCfg struct {
		Enabled           bool   `yaml:"Enable"`
		VerifyCertificate bool   `yaml:"VerifyCertificate"`
		CAFile            string `yaml:"CAFile"`
	}

	//LogStaticCfg contains the configuration for logging
	LogStaticCfg struct {
		LogLevel    int    `yaml:"LogLevel"`
		RitaLogPath string `yaml:"RitaLogPath"`
		LogToFile   bool   `yaml:"LogToFile"`
		LogToDB     bool   `yaml:"LogToDB"`
	}

	//BlacklistedStaticCfg is used to control the blacklisted analysis module
	BlacklistedStaticCfg struct {
		UseIPms            bool                  `yaml:"myIP.ms"`
		UseDNSBH           bool                  `yaml:"MalwareDomains.com"`
		UseMDL             bool                  `yaml:"MalwareDomainList.com"`
		SafeBrowsing       SafeBrowsingStaticCfg `yaml:"SafeBrowsing"`
		IPBlacklists       []string              `yaml:"CustomIPBlacklists"`
		HostnameBlacklists []string              `yaml:"CustomHostnameBlacklists"`
		URLBlacklists      []string              `yaml:"CustomURLBlacklists"`
	}

	//CrossrefStaticCfg is used to control the crossref analysis module
	CrossrefStaticCfg struct {
		BeaconThreshold float64 `yaml:"BeaconThreshold"`
	}

	//SafeBrowsingStaticCfg contains the details for contacting Google's safebrowsing api
	SafeBrowsingStaticCfg struct {
		APIKey   string `yaml:"APIKey"`
		Database string `yaml:"Database"`
	}

	//ScanningStaticCfg is used to control the scanning analysis module
	ScanningStaticCfg struct {
		ScanThreshold int `yaml:"ScanThreshold"`
	}

	//BeaconStaticCfg is used to control the beaconing analysis module
	BeaconStaticCfg struct {
		DefaultConnectionThresh int `yaml:"DefaultConnectionThresh"`
	}

	//BroStaticCfg controls the file parser
	BroStaticCfg struct {
		ImportDirectory string `yaml:"ImportDirectory"`
		DBRoot          string `yaml:"DBRoot"`
		MetaDB          string `yaml:"MetaDB"`
		ImportBuffer    int    `yaml:"ImportBuffer"`
	}
)

// loadStaticConfig attempts to parse a config file
func loadStaticConfig(cfgPath string) (*StaticCfg, error) {
	_, err := os.Stat(cfgPath)

	if os.IsNotExist(err) {
		return nil, err
	}

	cfgFile, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	return parseStaticConfig(cfgFile)
}

func parseStaticConfig(cfgFile []byte) (*StaticCfg, error) {
	var config = new(StaticCfg)
	err := yaml.Unmarshal(cfgFile, config)

	if err != nil {
		return config, err
	}

	// expand env variables, config is a pointer
	// so we have to call elem on the reflect value
	expandConfig(reflect.ValueOf(config).Elem())

	// set the socket time out in hours
	config.MongoDB.SocketTimeout *= time.Hour

	// clean all filepaths
	config.Log.RitaLogPath = filepath.Clean(config.Log.RitaLogPath)
	config.Blacklisted.SafeBrowsing.Database = filepath.Clean(config.Blacklisted.SafeBrowsing.Database)
	config.Bro.ImportDirectory = filepath.Clean(config.Bro.ImportDirectory)

	// grab the version constants set by the build process
	config.Version = Version
	config.ExactVersion = ExactVersion

	return config, nil
}
