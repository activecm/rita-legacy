package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/yaml.v2"
)

type (
	SystemConfig struct {
		LogType                 string         `yaml:"LogType"`
		GNUNetcatPath           string         `yaml:"GNUNetcatPath"`
		BaseInstallDirectory    string         `yaml:"BaseInstallDirectory"`
		BatchSize               int            `yaml:"BatchSize"`
		DB                      string         `yaml:"Database"`
		HostIntelDB             string         `yaml:"HostIntelDB"`
		ExternalHostsCollection string         `yaml:"ExternalHostsCollection"`
		DatabaseHost            string         `yaml:"DatabaseHost"`
		LogLevel                int            `yaml:"LogLevel"`
		Prefetch                float64        `yaml:"Prefetch"`
		Whitelist               []string       `yaml:"WhiteList"`
		BlacklistedConfig       BlacklistedCfg `yaml:"BlackListed"`
		DnsConfig               DnsCfg         `yaml:"Dns"`
		DurationConfig          DurationCfg    `yaml:"Duration"`
		ScanningConfig          ScanningCfg    `yaml:"Scanning"`
		StructureConfig         StructureCfg   `yaml:"Structure"`
		TBDConfig               TBDCfg         `yaml:"TBD"`
		UrlsConfig              UrlsCfg        `yaml:"Urls"`
		UserAgentConfig         UserAgentCfg   `yaml:"UserAgent"`
		BroConfig               BroCfg         `yaml:"Bro"`
	}

	StructureCfg struct {
		ConnTable       string `yaml:"ConnectionTable"`
		HttpTable       string `yaml:"HttpTable"`
		UniqueConnTable string `yaml:"UniqueConnectionTable"`
		HostTable       string `yaml:"HostTable"`
	}

	BlacklistedCfg struct {
		ThreadCount    int    `yaml:"ThreadCount"`
		ChannelSize    int    `yaml:"ChannelSize"`
		BlacklistTable string `yaml:"BlackListTable"`
	}

	DnsCfg struct {
		DnsTable string `yaml:"DnsTable"`
	}

	DurationCfg struct {
		DurationTimeScale string `yaml:"DurationTimeScale"`
	}

	ScanningCfg struct {
		ScanThreshold int    `yaml:"ScanThreshold"`
		ScanTable     string `yaml:"ScanTable"`
	}

	TBDCfg struct {
		DefaultBucketSize       float64 `yaml:"DefaultBucketSize"`
		DefaultConnectionThresh int     `yaml:"DefaultConnectionThresh"`
		TBDTable                string  `yaml:"TBDTable"`
	}

	UrlsCfg struct {
		UrlsTable      string `yaml:"UrlsTable"`
		HostnamesTable string `yaml:"HostnamesTable"`
	}

	UserAgentCfg struct {
		UserAgentTable string `yaml:"UserAgentTable"`
	}

	BroCfg struct {
		LogPath      string            `yaml:"LogPath"`
		DBPrefix     string            `yaml:"DBPrefix"`
		MetaDB       string            `yaml:"MetaDB"`
		BufferSize   int               `yaml:"BufferSize"`
		WriteThreads int               `yaml:"WriteThreads"`
		DirectoryMap map[string]string `yaml:"DirectoryMap"`
		FilesTable   string            `yaml:"FilesTable"`
		UseDates     bool              `yaml:"UseDates"`
	}

	// Resources provides a data structure for passing system Resources
	Resources struct {
		System  SystemConfig
		Session *mgo.Session
		Log     *log.Logger
	}
)

// CopySession allows systems to copy the resources session. Necessary for threading.
func (r *Resources) CopySession() *mgo.Session {
	return r.Session.Copy()
}

// LoadSystemConfig attempts to parse a config file
func LoadSystemConfig(cfgPath string) (SystemConfig, bool) {

	var config SystemConfig

	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		cfgFile, err := ioutil.ReadFile(cfgPath)
		if err != nil {
			return config, false
		}
		err = yaml.Unmarshal(cfgFile, &config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config: %s\n", err.Error())
			return config, false
		}
		return config, true
	}
	return config, false
}

// GetConfig retrieves a configuration in order of precedence
func GetConfig(cfgPath string) (SystemConfig, bool) {

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

// InitCofnig grabs the configuration file and intitializes the configuration data
// returnign a *Resources object which has all of the necessary configuration information
func InitConfig(cfgPath string) *Resources {

	config, ok := GetConfig(cfgPath)
	if !ok {
		fmt.Fprintf(os.Stdout, "Failed to config, exiting")
		os.Exit(-1)
	}

	// Fire up the logging system
	log, err := InitLog(config.LogLevel, config.LogType)
	if err != nil {
		fmt.Printf("Failed to prep logger: %s", err.Error())
		os.Exit(-1)
	}

	// Jump into the requested database
	session, err := mgo.Dial(config.DatabaseHost)
	if err != nil {
		fmt.Printf("Failed to connect to database: %s", err.Error(), config.DatabaseHost)
		os.Exit(-1)
	}

	return &Resources{Log: log, Session: session, System: config}
}

/*
 * Name:     InitLog
 * Purpose:  Initializes logging utilities
 * comments:
 */
func InitLog(level int, logtype string) (*log.Logger, error) {
	var logs = &log.Logger{}

	if logtype == "production" {
		logs.Formatter = new(log.JSONFormatter)
	} else {
		logs.Formatter = new(log.TextFormatter)
	}

	logs.Out = os.Stderr

	switch level {
	case 3:
		logs.Level = log.DebugLevel
		break
	case 2:
		logs.Level = log.InfoLevel
		break
	case 1:
		logs.Level = log.WarnLevel
		break
	case 0:
		logs.Level = log.ErrorLevel
	}

	return logs, nil
}
