package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"reflect"

	"gopkg.in/yaml.v2"
)

//VERSION is filled at compile time with the git version of RITA
var VERSION = "undefined"

type (
	//SystemConfig is the container for other config sections
	SystemConfig struct {
		BatchSize         int            `yaml:"BatchSize"`
		DatabaseHost      string         `yaml:"DatabaseHost"`
		Prefetch          float64        `yaml:"Prefetch"`
		LogConfig         LogCfg         `yaml:"LogConfig"`
		BlacklistedConfig BlacklistedCfg `yaml:"BlackListed"`
		DNSConfig         DNSCfg         `yaml:"Dns"`
		CrossrefConfig    CrossrefCfg    `yaml:"Crossref"`
		ScanningConfig    ScanningCfg    `yaml:"Scanning"`
		StructureConfig   StructureCfg   `yaml:"Structure"`
		BeaconConfig      BeaconCfg      `yaml:"Beacon"`
		UrlsConfig        UrlsCfg        `yaml:"Urls"`
		UserAgentConfig   UserAgentCfg   `yaml:"UserAgent"`
		BroConfig         BroCfg         `yaml:"Bro"`
		MetaTables        MetaCfg        `yaml:"MetaTables"`
		Version           string
	}

	//LogCfg contains the configuration for logging
	LogCfg struct {
		LogLevel     int    `yaml:"LogLevel"`
		RitaLogPath  string `yaml:"RitaLogPath"`
		LogToFile    bool   `yaml:"LogToFile"`
		RitaLogTable string `yaml:"RitaLogTable"`
		LogToDB      bool   `yaml:"LogToDB"`
	}

	//StructureCfg contains the names of the base level collections
	StructureCfg struct {
		ConnTable       string `yaml:"ConnectionTable"`
		HTTPTable       string `yaml:"HttpTable"`
		DNSTable        string `yaml:"DnsTable"`
		UniqueConnTable string `yaml:"UniqueConnectionTable"`
		HostTable       string `yaml:"HostTable"`
	}

	//BlacklistedCfg is used to control the blacklisted analysis module
	BlacklistedCfg struct {
		BlacklistDatabase  string          `yaml:"Database"`
		UseIPms            bool            `yaml:"myIP.ms"`
		UseDNSBH           bool            `yaml:"MalwareDomains.com"`
		UseMDL             bool            `yaml:"MalwareDomainList.com"`
		SafeBrowsing       SafeBrowsingCfg `yaml:"SafeBrowsing"`
		IPBlacklists       []string        `yaml:"CustomIPBlacklists"`
		HostnameBlacklists []string        `yaml:"CustomHostnameBlacklists"`
		URLBlacklists      []string        `yaml:"CustomURLBlacklists"`
		SourceIPsTable     string          `yaml:"SourceIPsTable"`
		DestIPsTable       string          `yaml:"DestIPsTable"`
		HostnamesTable     string          `yaml:"HostnamesTable"`
		UrlsTable          string          `yaml:"UrlsTable"`
	}

	//DNSCfg is used to control the dns analysis module
	DNSCfg struct {
		ExplodedDNSTable string `yaml:"ExplodedDnsTable"`
		HostnamesTable   string `yaml:"HostnamesTable"`
	}

	//CrossrefCfg is used to control the crossref analysis module
	CrossrefCfg struct {
		SourceTable     string  `yaml:"SourceTable"`
		DestTable       string  `yaml:"DestinationTable"`
		BeaconThreshold float64 `yaml:"BeaconThreshold"`
	}

	//SafeBrowsingCfg contains the details for contacting Google's safebrowsing api
	SafeBrowsingCfg struct {
		APIKey   string `yaml:"APIKey"`
		Database string `yaml:"Database"`
	}

	//ScanningCfg is used to control the scanning analysis module
	ScanningCfg struct {
		ScanThreshold int    `yaml:"ScanThreshold"`
		ScanTable     string `yaml:"ScanTable"`
	}

	//BeaconCfg is used to control the beaconing analysis module
	BeaconCfg struct {
		DefaultConnectionThresh int    `yaml:"DefaultConnectionThresh"`
		BeaconTable             string `yaml:"BeaconTable"`
	}

	//UrlsCfg is used to control the urls analysis module
	UrlsCfg struct {
		UrlsTable string `yaml:"UrlsTable"`
	}

	//UserAgentCfg is used to control the urls analysis module
	UserAgentCfg struct {
		UserAgentTable string `yaml:"UserAgentTable"`
	}

	//BroCfg controls the file parser
	BroCfg struct {
		LogPath         string            `yaml:"LogPath"`
		DBPrefix        string            `yaml:"DBPrefix"`
		MetaDB          string            `yaml:"MetaDB"`
		DirectoryMap    map[string]string `yaml:"DirectoryMap"`
		DefaultDatabase string            `yaml:"DefaultDatabase"`
		UseDates        bool              `yaml:"UseDates"`
		ImportBuffer    int               `yaml:"ImportBuffer"`
	}

	//MetaCfg contains the meta db collection names
	MetaCfg struct {
		FilesTable     string `yaml:"FilesTable"`
		DatabasesTable string `yaml:"DatabasesTable"`
	}
)

// GetConfig retrieves a configuration in order of precedence
func GetConfig(cfgPath string) (*SystemConfig, bool) {
	if cfgPath != "" {
		return loadSystemConfig(cfgPath)
	}

	// Get the user's homedir
	user, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get user info: %s\n", err.Error())
	} else {

		conf, ok := loadSystemConfig(user.HomeDir + "/.rita/config.yaml")
		if ok {
			return conf, ok
		}
	}

	// If none of the other configs have worked, go for the global config
	return loadSystemConfig("/etc/rita/config.yaml")
}

// loadSystemConfig attempts to parse a config file
func loadSystemConfig(cfgPath string) (*SystemConfig, bool) {
	var config = new(SystemConfig)

	config.Version = VERSION

	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		cfgFile, err := ioutil.ReadFile(cfgPath)
		if err != nil {
			return config, false
		}
		err = yaml.Unmarshal(cfgFile, config)

		// expand env variables, config is a pointer
		// so we have to call elem on the reflect value
		expandConfig(reflect.ValueOf(config).Elem())

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config: %s\n", err.Error())
			return config, false
		}
		return config, true
	}
	return config, false
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
