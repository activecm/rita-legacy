package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// Define the maximum value for the strobe connection limit
// This value must be low enough to ensure that documents that store
// arrays based off of connections will not grow beyond 16MB in size

// ex: beaconSNI has dat.tls.ts, dat.tls.bytes, dat.http.ts, and dat.http.bytes
// which are arrays of 64bit integers. Assuming that the size of an int64 in mongo
// is 8 bytes, then each of those arrays would require 16 bytes per connection
// The cummulative size of dat documents over all CIDs must fit within the document
// limit and allow enough room for other data within the document to also fit
// within ~16777216 bytes
const maxStrobeConnectionLimit int = 86400

// Define the minimum connection limit for beacon analysis. Currently we require a 24
// hour block of data for a dataset. Analyzing hosts that have fewer than at least one
// connection per hour could significantly increase both the analysis time and the number
// of false positives, so the threshold must be 23 or above.
const minBeaconConnectionThreshLimit int = 23

type (
	//StaticCfg is the container for other static config sections
	StaticCfg struct {
		UserConfig   UserCfgStaticCfg     `yaml:"UserConfig"`
		MongoDB      MongoDBStaticCfg     `yaml:"MongoDB"`
		Rolling      RollingStaticCfg     `yaml:"Rolling"`
		Log          LogStaticCfg         `yaml:"LogConfig"`
		Blacklisted  BlacklistedStaticCfg `yaml:"BlackListed"`
		Beacon       BeaconStaticCfg      `yaml:"Beacon"`
		BeaconProxy  BeaconProxyStaticCfg `yaml:"BeaconProxy"`
		BeaconSNI    BeaconSNIStaticCfg   `yaml:"BeaconSNI"`
		DNS          DNSStaticCfg         `yaml:"DNS"`
		UserAgent    UserAgentStaticCfg   `yaml:"UserAgent"`
		Bro          BroStaticCfg         `yaml:"Bro"` // kept in for MetaDB backwards compatibility
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
		MetaDB           string        `yaml:"MetaDB" default:"MetaDatabase"`
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
		MetaDB string `yaml:"MetaDB"` // kept in for backwards compatibility
	}

	//RollingStaticCfg controls the rolling database settings
	RollingStaticCfg struct {
		DefaultChunks int `yaml:"DefaultChunks" default:"24"`
		Rolling       bool
		CurrentChunk  int
		TotalChunks   int
	}

	//UserCfgStaticCfg contains
	UserCfgStaticCfg struct {
		UpdateCheckFrequency int `yaml:"UpdateCheckFrequency" default:"14"`
	}

	//BlacklistedStaticCfg is used to control the blacklisted analysis module
	BlacklistedStaticCfg struct {
		Enabled            bool     `yaml:"Enabled" default:"true"`
		UseFeodo           bool     `yaml:"feodotracker.abuse.ch" default:"true"`
		BlacklistDatabase  string   `yaml:"BlacklistDatabase" default:"rita-bl"`
		IPBlacklists       []string `yaml:"CustomIPBlacklists" default:"[]"`
		HostnameBlacklists []string `yaml:"CustomHostnameBlacklists" default:"[]"`
	}

	//BeaconStaticCfg is used to control the beaconing analysis module
	BeaconStaticCfg struct {
		Enabled                      bool    `yaml:"Enabled" default:"true"`
		DefaultConnectionThresh      int     `yaml:"DefaultConnectionThresh" default:"23"`
		TsWeight                     float64 `yaml:"TimestampScoreWeight" default:"0.25"`
		DsWeight                     float64 `yaml:"DatasizeScoreWeight" default:"0.25"`
		DurWeight                    float64 `yaml:"DurationScoreWeight" default:"0.25"`
		HistWeight                   float64 `yaml:"HistogramScoreWeight" default:"0.25"`
		DurMinHoursSeen              int     `yaml:"DurationMinHoursSeen" default:"6"`
		DurConsistencyIdealHoursSeen int     `yaml:"DurationConsistencyIdealHoursSeen" default:"12"`
		HistBimodalBucketSize        float64 `yaml:"HistogramBimodalBucketSize" default:"0.05"`
		HistBimodalOutlierRemoval    int     `yaml:"HistogramBimodalOutlierRemoval" default:"1"`
		HistBimodalMinHoursSeen      int     `yaml:"HistogramBimodalMinHoursSeen" default:"11"`
	}

	//BeaconProxyStaticCfg is used to control the proxy beaconing analysis module
	BeaconProxyStaticCfg struct {
		Enabled                      bool    `yaml:"Enabled" default:"true"`
		DefaultConnectionThresh      int     `yaml:"DefaultConnectionThresh" default:"23"`
		TsWeight                     float64 `yaml:"TimestampScoreWeight" default:"0.333"`
		DurWeight                    float64 `yaml:"DurationScoreWeight" default:"0.333"`
		HistWeight                   float64 `yaml:"HistogramScoreWeight" default:"0.333"`
		DurMinHoursSeen              int     `yaml:"DurationMinHoursSeen" default:"6"`
		DurConsistencyIdealHoursSeen int     `yaml:"DurationConsistencyIdealHoursSeen" default:"12"`
		HistBimodalBucketSize        float64 `yaml:"HistogramBimodalBucketSize" default:"0.05"`
		HistBimodalOutlierRemoval    int     `yaml:"HistogramBimodalOutlierRemoval" default:"1"`
		HistBimodalMinHoursSeen      int     `yaml:"HistogramBimodalMinHoursSeen" default:"11"`
	}

	//BeaconSNIStaticCfg is used to control the SNI beaconing analysis module
	BeaconSNIStaticCfg struct {
		Enabled                      bool    `yaml:"Enabled" default:"true"`
		DefaultConnectionThresh      int     `yaml:"DefaultConnectionThresh" default:"23"`
		TsWeight                     float64 `yaml:"TimestampScoreWeight" default:"0.25"`
		DsWeight                     float64 `yaml:"DatasizeScoreWeight" default:"0.25"`
		DurWeight                    float64 `yaml:"DurationScoreWeight" default:"0.25"`
		HistWeight                   float64 `yaml:"HistogramScoreWeight" default:"0.25"`
		DurMinHoursSeen              int     `yaml:"DurationMinHoursSeen" default:"6"`
		DurConsistencyIdealHoursSeen int     `yaml:"DurationConsistencyIdealHoursSeen" default:"12"`
		HistBimodalBucketSize        float64 `yaml:"HistogramBimodalBucketSize" default:"0.05"`
		HistBimodalOutlierRemoval    int     `yaml:"HistogramBimodalOutlierRemoval" default:"1"`
		HistBimodalMinHoursSeen      int     `yaml:"HistogramBimodalMinHoursSeen" default:"11"`
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
		AlwaysInclude            []string `yaml:"AlwaysInclude" default:"[]"`
		NeverInclude             []string `yaml:"NeverInclude" default:"[\"0.0.0.0/32\", \"127.0.0.0/8\", \"169.254.0.0/16\", \"224.0.0.0/4\", \"255.255.255.255/32\", \"::1/128\", \"fe80::/10\", \"ff00::/8\"]"`
		InternalSubnets          []string `yaml:"InternalSubnets" default:"[\"10.0.0.0/8\", \"172.16.0.0/12\", \"192.168.0.0/16\"]"`
		AlwaysIncludeDomain      []string `yaml:"AlwaysIncludeDomain" default:"[]"`
		NeverIncludeDomain       []string `yaml:"NeverIncludeDomain" default:"[]"`
		FilterExternalToInternal bool     `yaml:"FilterExternalToInternal" default:"true"`
	}

	//StrobeStaticCfg controls the maximum number of connections between any two given hosts
	StrobeStaticCfg struct {
		ConnectionLimit int `yaml:"ConnectionLimit" default:"86400"`
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

	// migrate MetaDB entry from old location (Bro:MetaDB) if there is a value in the
	// old location and the new location (MongoDB:MetaDB) is still the default (MetaDatabase)
	if config.Bro.MetaDB != "" && config.MongoDB.MetaDB == "MetaDatabase" {
		config.MongoDB.MetaDB = config.Bro.MetaDB
	}

	// limit the strobe connection limit to the maximum allowed
	if config.Strobe.ConnectionLimit > maxStrobeConnectionLimit {
		config.Strobe.ConnectionLimit = maxStrobeConnectionLimit
	}

	// limit the beacon threshold to the minimum allowed
	if config.Beacon.DefaultConnectionThresh < minBeaconConnectionThreshLimit {
		config.Beacon.DefaultConnectionThresh = minBeaconConnectionThreshLimit
	}

	// limit the beacon sni threshold to the minimum allowed
	if config.BeaconSNI.DefaultConnectionThresh < minBeaconConnectionThreshLimit {
		config.BeaconSNI.DefaultConnectionThresh = minBeaconConnectionThreshLimit
	}

	// limit the beacon proxy threshold to the minimum allowed
	if config.BeaconProxy.DefaultConnectionThresh < minBeaconConnectionThreshLimit {
		config.BeaconProxy.DefaultConnectionThresh = minBeaconConnectionThreshLimit
	}

	// make sure value is above zero to avoid division by zero
	if config.Beacon.DurConsistencyIdealHoursSeen < 1 {
		config.Beacon.DurConsistencyIdealHoursSeen = 1
	}
	if config.BeaconProxy.DurConsistencyIdealHoursSeen < 1 {
		config.BeaconProxy.DurConsistencyIdealHoursSeen = 1
	}
	if config.BeaconSNI.DurConsistencyIdealHoursSeen < 1 {
		config.BeaconSNI.DurConsistencyIdealHoursSeen = 1
	}

	// expand env variables, config is a pointer
	// so we have to call elem on the reflect value
	expandConfig(reflect.ValueOf(config).Elem())

	// set the socket time out in hours
	config.MongoDB.SocketTimeout *= time.Hour

	// clean all filepaths
	config.Log.RitaLogPath = filepath.Clean(config.Log.RitaLogPath)

	// grab the version constants set by the build process
	config.Version = Version
	config.ExactVersion = ExactVersion

	return nil
}
