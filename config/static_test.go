package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const staticConfigParserTestConfig = `
MongoDB:
    ConnectionString: mongodb://localhost:27017
    AuthenticationMechanism: null
    SocketTimeout: 2
    TLS:
        Enable: false
        VerifyCertificate: false
        CAFile: aaaaa
LogConfig:
    LogLevel: 2
    RitaLogPath: /var/lib/rita/logs
    LogToFile: true
    LogToDB: true
Bro:
    ImportDirectory: /opt/bro/logs/
    DBRoot: "RITA"
    MetaDB: MetaDatabase
    ImportBuffer: 100000
UserConfig:
    UpdateCheckFrequency: 14
BlackListed:
    myIP.ms: true
    MalwareDomains.com: true
    MalwareDomainList.com: true
    CustomIPBlacklists: [test1]
    CustomHostnameBlacklists: [test2]
Beacon:
    DefaultConnectionThresh: 24
Strobe:
    ConnectionLimit: 250000
Filtering:
    AlwaysInclude: [8.8.8.8/32]
    InternalSubnets: [10.0.0.0/8,172.16.0.0/12,192.168.0.0/16]
`

var testConfigFullExp = StaticCfg{
	MongoDB: MongoDBStaticCfg{
		ConnectionString: "mongodb://localhost:27017",
		AuthMechanism:    "",
		SocketTimeout:    2 * time.Hour,
		TLS: TLSStaticCfg{
			Enabled:           false,
			VerifyCertificate: false,
			CAFile:            "aaaaa",
		},
	},
	Log: LogStaticCfg{
		LogLevel:    2,
		RitaLogPath: "/var/lib/rita/logs",
		LogToFile:   true,
		LogToDB:     true,
	},
	Bro: BroStaticCfg{
		ImportDirectory: "/opt/bro/logs",
		DBRoot:          "RITA",
		MetaDB:          "MetaDatabase",
		ImportBuffer:    100000,
	},
	UserConfig: UserCfgStaticCfg{
		UpdateCheckFrequency: 14,
	},
	Blacklisted: BlacklistedStaticCfg{
		UseIPms:            true,
		UseMDL:             true,
		UseDNSBH:           true,
		IPBlacklists:       []string{"test1"},
		HostnameBlacklists: []string{"test2"},
	},
	Beacon: BeaconStaticCfg{
		DefaultConnectionThresh: 24,
	},
	Strobe: StrobeStaticCfg{
		ConnectionLimit: 250000,
	},
	Filtering: FilteringStaticCfg{
		AlwaysInclude:   []string{"8.8.8.8/32"},
		InternalSubnets: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
	},
}

// TestParseStaticConfig ensures that a yaml config
// string is correctly converted into a StaticCfg struct.
func TestParseStaticConfig(t *testing.T) {
	config := &StaticCfg{}
	err := parseStaticConfig([]byte(staticConfigParserTestConfig), config)

	// We are not testing the version setting ensure they are equal
	testConfigFullExp.Version = config.Version
	testConfigFullExp.ExactVersion = config.ExactVersion

	assert.Nil(t, err)
	assert.Equal(t, *config, testConfigFullExp)
}

// TestFilePathCleaning ensures that paths specified
// in a config file are cleaned up correctly.
func TestFilePathCleaning(t *testing.T) {
	testConfig := `
LogConfig:
    RitaLogPath: /var/lib/rita/incorrect/./../logs/
Bro:
    ImportDirectory: /opt/bro/incorrect/./../../bro/logs/
`
	testConfigExp := StaticCfg{
		Log: LogStaticCfg{
			RitaLogPath: "/var/lib/rita/logs",
		},
		Bro: BroStaticCfg{
			ImportDirectory: "/opt/bro/logs",
		},
	}
	config := &StaticCfg{}
	err := parseStaticConfig([]byte(testConfig), config)

	// We are not testing the version setting ensure they are equal
	testConfigExp.Version = config.Version
	testConfigExp.ExactVersion = config.ExactVersion

	assert.Nil(t, err)
	assert.Equal(t, *config, testConfigExp)
}
