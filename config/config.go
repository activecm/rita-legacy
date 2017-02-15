package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"

	"gopkg.in/yaml.v2"
)

type (
	SystemConfig struct {
		LogType           string          `yaml:"LogType"`
		GNUNetcatPath     string          `yaml:"GNUNetcatPath"`
		BatchSize         int             `yaml:"BatchSize"`
		DatabaseHost      string          `yaml:"DatabaseHost"`
		LogLevel          int             `yaml:"LogLevel"`
		Prefetch          float64         `yaml:"Prefetch"`
		Whitelist         []string        `yaml:"WhiteList"`
		ImportWhitelist   bool            `yaml:"ImportWhitelist"`
		BlacklistedConfig BlacklistedCfg  `yaml:"BlackListed"`
		CrossrefConfig    CrossrefCfg     `yaml:"Crossref"`
		ScanningConfig    ScanningCfg     `yaml:"Scanning"`
		StructureConfig   StructureCfg    `yaml:"Structure"`
		TBDConfig         TBDCfg          `yaml:"TBD"`
		UrlsConfig        UrlsCfg         `yaml:"Urls"`
		UserAgentConfig   UserAgentCfg    `yaml:"UserAgent"`
		BroConfig         BroCfg          `yaml:"Bro"`
		SafeBrowsing      SafeBrowsingCfg `yaml:"SafeBrowsing"`
	}

	StructureCfg struct {
		ConnTable       string `yaml:"ConnectionTable"`
		HttpTable       string `yaml:"HttpTable"`
		DnsTable        string `yaml:"DnsTable"`
		UniqueConnTable string `yaml:"UniqueConnectionTable"`
		HostTable       string `yaml:"HostTable"`
	}

	BlacklistedCfg struct {
		ThreadCount       int    `yaml:"ThreadCount"`
		ChannelSize       int    `yaml:"ChannelSize"`
		BlacklistTable    string `yaml:"BlackListTable"`
		BlacklistDatabase string `yaml:"Database"`
	}

	CrossrefCfg struct {
		InternalTable string  `yaml:"InternalTable"`
		ExternalTable string  `yaml:"ExternalTable"`
		TBDThreshold  float64 `yaml:"TBDThreshold"`
	}

	SafeBrowsingCfg struct {
		APIKey   string `yaml:"APIKey"`
		Database string `yaml:"Database"`
	}

	ScanningCfg struct {
		ScanThreshold int    `yaml:"ScanThreshold"`
		ScanTable     string `yaml:"ScanTable"`
	}

	TBDCfg struct {
		DefaultConnectionThresh int    `yaml:"DefaultConnectionThresh"`
		TBDTable                string `yaml:"TBDTable"`
	}

	UrlsCfg struct {
		UrlsTable      string `yaml:"UrlsTable"`
		HostnamesTable string `yaml:"HostnamesTable"`
	}

	UserAgentCfg struct {
		UserAgentTable string `yaml:"UserAgentTable"`
	}

	BroCfg struct {
		LogPath         string            `yaml:"LogPath"`
		DBPrefix        string            `yaml:"DBPrefix"`
		MetaDB          string            `yaml:"MetaDB"`
		WriteThreads    int               `yaml:"WriteThreads"`
		DirectoryMap    map[string]string `yaml:"DirectoryMap"`
		DefaultDatabase string            `yaml:"DefaultDatabase"`
		UseDates        bool              `yaml:"UseDates"`
	}
)

// LoadSystemConfig attempts to parse a config file
func LoadSystemConfig(cfgPath string) (*SystemConfig, bool) {
	var config = new(SystemConfig)

	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		cfgFile, err := ioutil.ReadFile(cfgPath)
		if err != nil {
			return config, false
		}
		err = yaml.Unmarshal(cfgFile, config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config: %s\n", err.Error())
			return config, false
		}
		return config, true
	}
	return config, false
}

// GetConfig retrieves a configuration in order of precedence
func GetConfig(cfgPath string) (*SystemConfig, bool) {
	if cfgPath != "" {
		return LoadSystemConfig(cfgPath)
	}

	// Get the user's homedir
	user, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get user info: %s\n", err.Error())
	} else {

		conf, ok := LoadSystemConfig(user.HomeDir + "/.rita")
		if ok {
			return conf, ok
		}
	}

	// If none of the other configs have worked, go for the homedir config
	return LoadSystemConfig("/etc/rita/config.yaml")
}
