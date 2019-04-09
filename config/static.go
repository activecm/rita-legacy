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
		Beacon       BeaconStaticCfg      `yaml:"Beacon"`
		DNS          DNSStaticCfg         `yaml:"DNS"`
		UserAgent    UserAgentStaticCfg   `yaml:"UserAgent"`
		Bro          BroStaticCfg         `yaml:"Bro"`
		Filtering    FilteringStaticCfg   `yaml:"Filtering"`
		Strobe       StrobeStaticCfg      `yaml:"Strobe"`
		Version      string
		ExactVersion string
	}

	//MongoDBStaticCfg contains the means for connecting to MongoDB
	MongoDBStaticCfg struct {
		ConnectionString string        `yaml:"ConnectionString" default:"mongodb://localhost:27017"`
		AuthMechanism    string        `yaml:"AuthenticationMechanism" default:""`
		SocketTimeout    time.Duration `yaml:"SocketTimeout" default:"2"`
		TLS              TLSStaticCfg  `yaml:"TLS"`
	}

	//TLSStaticCfg contains the means for connecting to MongoDB over TLS
	TLSStaticCfg struct {
		Enabled           bool   `yaml:"Enable" default:"false"`
		VerifyCertificate bool   `yaml:"VerifyCertificate" default:"false"`
		CAFile            string `yaml:"CAFile" default:""`
	}

	//LogStaticCfg contains the configuration for logging
	LogStaticCfg struct {
		LogLevel    int    `yaml:"LogLevel" default:"2"`
		RitaLogPath string `yaml:"RitaLogPath" default:"/var/lib/rita/logs"`
		LogToFile   bool   `yaml:"LogToFile" default:"true"`
		LogToDB     bool   `yaml:"LogToDB" default:"true"`
	}

	//BroStaticCfg controls the file parser
	BroStaticCfg struct {
		ImportDirectory string `yaml:"ImportDirectory" default:"/opt/bro/logs/"`
		DBName          string `yaml:"DBName" default:"RITA"`
		MetaDB          string `yaml:"MetaDB" default:"MetaDatabase"`
		ImportBuffer    int    `yaml:"ImportBuffer" default:"30000"`
		Rolling         bool
		TotalChunks     int
		CurrentChunk    int
	}

	//UserCfgStaticCfg contains
	UserCfgStaticCfg struct {
		UpdateCheckFrequency int `yaml:"UpdateCheckFrequency" default:"14"`
	}

	//BlacklistedStaticCfg is used to control the blacklisted analysis module
	BlacklistedStaticCfg struct {
		Enabled            bool     `yaml:"Enabled" default:"true"`
		UseIPms            bool     `yaml:"myIP.ms" default:"true"`
		UseDNSBH           bool     `yaml:"MalwareDomains.com" default:"true"`
		UseMDL             bool     `yaml:"MalwareDomainList.com" default:"true"`
		BlacklistDatabase  string   `yaml:"BlacklistDatabase" default:"rita-bl"`
		IPBlacklists       []string `yaml:"CustomIPBlacklists" default:"[]"`
		HostnameBlacklists []string `yaml:"CustomHostnameBlacklists" default:"[]"`
	}

	//BeaconStaticCfg is used to control the beaconing analysis module
	BeaconStaticCfg struct {
		Enabled                 bool `yaml:"Enabled" default:"true"`
		DefaultConnectionThresh int  `yaml:"DefaultConnectionThresh" default:"20"`
	}

	//DNSStaticCfg is used to control the DNS analysis module
	DNSStaticCfg struct {
		Enabled bool `yaml:"Enabled" default:"true"`
	}

	//UserAgentStaticCfg is used to control the User Agent analysis module
	UserAgentStaticCfg struct {
		Enabled bool `yaml:"Enabled" default:"true"`
	}

	//FilteringStaticCfg controls address filtering
	FilteringStaticCfg struct {
		AlwaysInclude   []string `yaml:"AlwaysInclude" default:"[]"`
		NeverInclude    []string `yaml:"NeverInclude" default:"[]"`
		InternalSubnets []string `yaml:"InternalSubnets" default:"[]"`
	}

	//StrobeStaticCfg controls the maximum number of connections between any two given hosts
	StrobeStaticCfg struct {
		ConnectionLimit int `yaml:"ConnectionLimit" default:"250000"`
	}
)

// readStaticConfigFile attempts to read the contents of the
// given cfgPath file path (e.g. /etc/rita/config.yaml)
func readStaticConfigFile(cfgPath string) ([]byte, error) {
	_, err := os.Stat(cfgPath)

	if os.IsNotExist(err) {
		return nil, err
	}

	cfgFile, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	return cfgFile, nil
}

// parseStaticConfig loads the yaml from cfgFile into the provided config struct.
// It also fixes up misc values that need tweaking into the right format.
func parseStaticConfig(cfgFile []byte, config *StaticCfg) error {
	err := yaml.Unmarshal(cfgFile, config)

	if err != nil {
		return err
	}

	// expand env variables, config is a pointer
	// so we have to call elem on the reflect value
	expandConfig(reflect.ValueOf(config).Elem())

	// set the socket time out in hours
	config.MongoDB.SocketTimeout *= time.Hour

	// clean all filepaths
	config.Log.RitaLogPath = filepath.Clean(config.Log.RitaLogPath)
	config.Bro.ImportDirectory = filepath.Clean(config.Bro.ImportDirectory)

	// grab the version constants set by the build process
	config.Version = Version
	config.ExactVersion = ExactVersion

	return nil
}
