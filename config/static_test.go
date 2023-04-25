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
    MetaDB: MetaDatabase
LogConfig:
    LogLevel: 2
    RitaLogPath: /var/lib/rita/logs
    LogToFile: true
    LogToDB: true
UserConfig:
    UpdateCheckFrequency: 14
BlackListed:
    feodotracker.abuse.ch: true
    BlacklistDatabase: "rita-bl"
    CustomIPBlacklists: [test1]
    CustomHostnameBlacklists: [test2]
DNS:
    Enabled: true
Beacon:
    Enabled: true
    DefaultConnectionThresh: 5
    TimestampScoreWeight: 0.25
    DatasizeScoreWeight: 0.25
    DurationScoreWeight: 0.25
    HistogramScoreWeight: 0.25
    DurationMinHoursSeen: 6
    DurationConsistencyIdealHoursSeen: 12
    HistogramBimodalBucketSize: 0.05
    HistogramBimodalOutlierRemoval: 1
    HistogramBimodalMinHoursSeen: 11
BeaconSNI:
    Enabled: true
    DefaultConnectionThresh: 5
    TimestampScoreWeight: 0.25
    DatasizeScoreWeight: 0.25
    DurationScoreWeight: 0.25
    HistogramScoreWeight: 0.25
    DurationMinHoursSeen: 6
    DurationConsistencyIdealHoursSeen: 12
    HistogramBimodalBucketSize: 0.05
    HistogramBimodalOutlierRemoval: 1
    HistogramBimodalMinHoursSeen: 11
BeaconProxy:
    Enabled: true
    DefaultConnectionThresh: 5
    TimestampScoreWeight: 0.333
    DurationScoreWeight: 0.333
    HistogramScoreWeight: 0.333
    DurationMinHoursSeen: 6
    DurationConsistencyIdealHoursSeen: 12
    HistogramBimodalBucketSize: 0.05
    HistogramBimodalOutlierRemoval: 1
    HistogramBimodalMinHoursSeen: 11
Strobe:
    ConnectionLimit: 250000
Filtering:
    AlwaysInclude: ["8.8.8.8/32"]
    NeverInclude: ["8.8.4.4/32"]
    InternalSubnets: ["10.0.0.0/8","172.16.0.0/12","192.168.0.0/16"]
    AlwaysIncludeDomain: ["bad.com", "google.com", "*.myotherdomain.com"]
    NeverIncludeDomain: ["good.com", "google.com", "*.mydomain.com"]
    FilterExternalToInternal: true
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
		MetaDB: "MetaDatabase",
	},
	Log: LogStaticCfg{
		LogLevel:    2,
		RitaLogPath: "/var/lib/rita/logs",
		LogToFile:   true,
		LogToDB:     true,
	},
	UserConfig: UserCfgStaticCfg{
		UpdateCheckFrequency: 14,
	},
	Blacklisted: BlacklistedStaticCfg{
		UseFeodo:           true,
		BlacklistDatabase:  "rita-bl",
		IPBlacklists:       []string{"test1"},
		HostnameBlacklists: []string{"test2"},
	},
	DNS: DNSStaticCfg{
		Enabled: true,
	},
	Beacon: BeaconStaticCfg{
		Enabled:                      true,
		DefaultConnectionThresh:      minBeaconConnectionThreshLimit,
		TsWeight:                     0.25,
		DsWeight:                     0.25,
		DurWeight:                    0.25,
		HistWeight:                   0.25,
		DurMinHoursSeen:              6,
		DurConsistencyIdealHoursSeen: 12,
		HistBimodalBucketSize:        0.05,
		HistBimodalOutlierRemoval:    1,
		HistBimodalMinHoursSeen:      11,
	},
	BeaconSNI: BeaconSNIStaticCfg{
		Enabled:                      true,
		DefaultConnectionThresh:      minBeaconConnectionThreshLimit,
		TsWeight:                     0.25,
		DsWeight:                     0.25,
		DurWeight:                    0.25,
		HistWeight:                   0.25,
		DurMinHoursSeen:              6,
		DurConsistencyIdealHoursSeen: 12,
		HistBimodalBucketSize:        0.05,
		HistBimodalOutlierRemoval:    1,
		HistBimodalMinHoursSeen:      11,
	},
	BeaconProxy: BeaconProxyStaticCfg{
		Enabled:                      true,
		DefaultConnectionThresh:      minBeaconConnectionThreshLimit,
		TsWeight:                     0.333,
		DurWeight:                    0.333,
		HistWeight:                   0.333,
		DurMinHoursSeen:              6,
		DurConsistencyIdealHoursSeen: 12,
		HistBimodalBucketSize:        0.05,
		HistBimodalOutlierRemoval:    1,
		HistBimodalMinHoursSeen:      11,
	},
	Strobe: StrobeStaticCfg{
		ConnectionLimit: maxStrobeConnectionLimit,
	},
	Filtering: FilteringStaticCfg{
		AlwaysInclude:            []string{"8.8.8.8/32"},
		NeverInclude:             []string{"8.8.4.4/32"},
		InternalSubnets:          []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
		AlwaysIncludeDomain:      []string{"bad.com", "google.com", "*.myotherdomain.com"},
		NeverIncludeDomain:       []string{"good.com", "google.com", "*.mydomain.com"},
		FilterExternalToInternal: true,
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
`
	testConfigExp := StaticCfg{
		Log: LogStaticCfg{
			RitaLogPath: "/var/lib/rita/logs",
		},
	}
	config := &StaticCfg{}
	err := parseStaticConfig([]byte(testConfig), config)

	// We are not testing the version setting ensure they are equal
	testConfigExp.Version = config.Version
	testConfigExp.ExactVersion = config.ExactVersion

	assert.Nil(t, err)
	assert.Equal(t, config.Log, testConfigExp.Log)
}
