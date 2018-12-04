package config

const testConfig = `
MongoDB:
    ConnectionString: null
    AuthenticationMechanism: null
    SocketTimeout: 2
    TLS:
        Enable: false
        VerifyCertificate: false
        CAFile: null
LogConfig:
    LogLevel: 3
    RitaLogPath: null
    LogToFile: false
    LogToDB: true
Bro:
    ImportDirectory: null
    DBRoot: RITA-TEST
    MetaDB: RITA-TEST-MetaDatabase
    ImportBuffer: 100000
BlackListed:
    myIP.ms: false
    MalwareDomains.com: false
    MalwareDomainList.com: false
    CustomIPBlacklists: []
    CustomHostnameBlacklists: []
Crossref:
    BeaconThreshold: .7
Beacon:
    DefaultConnectionThresh: 24
`

// LoadTestingConfig loads the hard coded testing config
func LoadTestingConfig(mongoURI string) (*Config, error) {
	Version = "v0.0.0+testing"
	ExactVersion = "v0.0.0+testing"
	var config = new(Config)
	static, err := parseStaticConfig([]byte(testConfig))
	if err != nil {
		return config, err
	}
	config.S = *static

	config.S.MongoDB.ConnectionString = mongoURI

	config.T = *loadTableConfig()

	running, err := loadRunningConfig(static)
	if err != nil {
		return config, err
	}
	config.R = *running

	return config, nil
}
