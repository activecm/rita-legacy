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
    SafeBrowsing:
        APIKey: "asdf"
        Database: /var/lib/rita/safebrowsing
    CustomIPBlacklists: [test1]
    CustomHostnameBlacklists: [test2]
    CustomURLBlacklists: [test3]
Crossref:
    BeaconThreshold: .7
Scanning:
    ScanThreshold: 50
Beacon:
    DefaultConnectionThresh: 24
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
		UseIPms:  true,
		UseMDL:   true,
		UseDNSBH: true,
		SafeBrowsing: SafeBrowsingStaticCfg{
			APIKey:   "asdf",
			Database: "/var/lib/rita/safebrowsing",
		},
		IPBlacklists:       []string{"test1"},
		HostnameBlacklists: []string{"test2"},
		URLBlacklists:      []string{"test3"},
	},
	Crossref: CrossrefStaticCfg{
		BeaconThreshold: .7,
	},
	Scanning: ScanningStaticCfg{
		ScanThreshold: 50,
	},
	Beacon: BeaconStaticCfg{
		DefaultConnectionThresh: 24,
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
BlackListed:
    SafeBrowsing:
        Database: /var/lib/rita/incorrect/./../safebrowsing
`
	testConfigExp := StaticCfg{
		Log: LogStaticCfg{
			RitaLogPath: "/var/lib/rita/logs",
		},
		Blacklisted: BlacklistedStaticCfg{
			SafeBrowsing: SafeBrowsingStaticCfg{
				Database: "/var/lib/rita/safebrowsing",
			},
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
