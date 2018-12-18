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
BlackListed:
    myIP.ms: true
    MalwareDomains.com: true
    MalwareDomainList.com: true
		BlacklistDatabase: "rita-bl"
    CustomIPBlacklists: [test1]
    CustomHostnameBlacklists: [test2]
Beacon:
    DefaultConnectionThresh: 24
Filtering:
    AlwaysInclude: ["8.8.8.8/32"]
    InternalSubnets: ["10.0.0.0/8","172.16.0.0/12","192.168.0.0/16"]

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
	Blacklisted: BlacklistedStaticCfg{
		UseIPms:            true,
		UseMDL:             true,
		UseDNSBH:           true,
		BlacklistDatabase:  "rita-bl",
		IPBlacklists:       []string{"test1"},
		HostnameBlacklists: []string{"test2"},
	},
	Beacon: BeaconStaticCfg{
		DefaultConnectionThresh: 24,
	},
	Filtering: FilteringStaticCfg{
		AlwaysInclude:   []string{"8.8.8.8/32"},
		InternalSubnets: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
	},
}

func TestParseStaticConfig(t *testing.T) {
	config, err := parseStaticConfig([]byte(staticConfigParserTestConfig))
	testConfigFullExp.Version = config.Version
	testConfigFullExp.ExactVersion = config.ExactVersion
	assert.Nil(t, err)
	assert.Equal(t, *config, testConfigFullExp)
}

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
	config, err := parseStaticConfig([]byte(testConfig))
	testConfigExp.Version = config.Version
	testConfigExp.ExactVersion = config.ExactVersion
	assert.Nil(t, err)
	assert.Equal(t, *config, testConfigExp)
}
