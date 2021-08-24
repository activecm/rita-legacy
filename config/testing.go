package config

import (
	"github.com/creasty/defaults"
)

const testConfig = `
MongoDB:
    ConnectionString: null
    AuthenticationMechanism: null
    SocketTimeout: 2
    TLS:
        Enable: false
        VerifyCertificate: false
        CAFile: null
    MetaDB: RITA-TEST-MetaDatabase
LogConfig:
    LogLevel: 3
    RitaLogPath: null
    LogToFile: false
    LogToDB: true
UserConfig:
    UpdateCheckFrequency: 14
BlackListed:
    myIP.ms: false
    MalwareDomains.com: false
    MalwareDomainList.com: false
    CustomIPBlacklists: []
    CustomHostnameBlacklists: []
DNS:
    Enabled: true
Beacon:
    Enabled: true
    DefaultConnectionThresh: 24
BeaconFQDN:
    Enabled: true
    DefaultConnectionThresh: 24
BeaconProxy:
    Enabled: true
    DefaultConnectionThresh: 24
Strobe:
    ConnectionLimit: 250000
Filtering:
    AlwaysInclude: ["8.8.8.8/32"]
    NeverInclude: ["8.8.4.4/32"]
    InternalSubnets: ["10.0.0.0/8","172.16.0.0/12","192.168.0.0/16"]
    HTTPProxyServers: ["1.1.1.1", "1.1.1.2/32", "1.2.0.0/16"]
    AlwaysIncludeDomain: ["bad.com", "google.com", "*.myotherdomain.com"]
    NeverIncludeDomain: ["good.com", "google.com", "*.mydomain.com"]
    FilterExternalToInternal: false
`

// LoadTestingConfig loads the hard coded testing config
func LoadTestingConfig(mongoURI string) (*Config, error) {
	config := &Config{}

	// Initialize table config to the default values
	if err := defaults.Set(&config.T); err != nil {
		return nil, err
	}

	// Initialize static config to the default values
	if err := defaults.Set(&config.S); err != nil {
		return nil, err
	}

	config.S.MongoDB.ConnectionString = mongoURI

	// Deserialize the yaml file contents into the static config
	if err := parseStaticConfig([]byte(testConfig), &config.S); err != nil {
		return nil, err
	}

	config.S.Version = "v0.0.0+testing"
	config.S.ExactVersion = "v0.0.0+testing"

	// Use the static config to initialize the running config
	if err := initRunningConfig(&config.S, &config.R); err != nil {
		return nil, err
	}

	return config, nil
}
