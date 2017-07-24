package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"

	yaml "gopkg.in/yaml.v2"
)

type (
	//TableCfg is the container for other table config sections
	TableCfg struct {
		Log         LogTableCfg         `yaml:"LogConfig"`
		Blacklisted BlacklistedTableCfg `yaml:"BlackListed"`
		DNS         DNSTableCfg         `yaml:"Dns"`
		Crossref    CrossrefTableCfg    `yaml:"Crossref"`
		Scanning    ScanningTableCfg    `yaml:"Scanning"`
		Structure   StructureTableCfg   `yaml:"Structure"`
		Beacon      BeaconTableCfg      `yaml:"Beacon"`
		Urls        UrlsTableCfg        `yaml:"Urls"`
		UserAgent   UserAgentTableCfg   `yaml:"UserAgent"`
		Meta        MetaTableCfg        `yaml:"MetaTables"`
	}

	//LogTableCfg contains the configuration for logging
	LogTableCfg struct {
		RitaLogTable string `yaml:"RitaLogTable"`
	}

	//StructureTableCfg contains the names of the base level collections
	StructureTableCfg struct {
		ConnTable       string `yaml:"ConnectionTable"`
		HTTPTable       string `yaml:"HttpTable"`
		DNSTable        string `yaml:"DnsTable"`
		UniqueConnTable string `yaml:"UniqueConnectionTable"`
		HostTable       string `yaml:"HostTable"`
	}

	//BlacklistedTableCfg is used to control the blacklisted analysis module
	BlacklistedTableCfg struct {
		BlacklistDatabase string `yaml:"Database"`
		SourceIPsTable    string `yaml:"SourceIPsTable"`
		DestIPsTable      string `yaml:"DestIPsTable"`
		HostnamesTable    string `yaml:"HostnamesTable"`
		UrlsTable         string `yaml:"UrlsTable"`
	}

	//DNSTableCfg is used to control the dns analysis module
	DNSTableCfg struct {
		ExplodedDNSTable string `yaml:"ExplodedDnsTable"`
		HostnamesTable   string `yaml:"HostnamesTable"`
	}

	//CrossrefTableCfg is used to control the crossref analysis module
	CrossrefTableCfg struct {
		SourceTable string `yaml:"SourceTable"`
		DestTable   string `yaml:"DestinationTable"`
	}

	//ScanningTableCfg is used to control the scanning analysis module
	ScanningTableCfg struct {
		ScanTable string `yaml:"ScanTable"`
	}

	//BeaconTableCfg is used to control the beaconing analysis module
	BeaconTableCfg struct {
		BeaconTable string `yaml:"BeaconTable"`
	}

	//UrlsTableCfg is used to control the urls analysis module
	UrlsTableCfg struct {
		UrlsTable string `yaml:"UrlsTable"`
	}

	//UserAgentTableCfg is used to control the urls analysis module
	UserAgentTableCfg struct {
		UserAgentTable string `yaml:"UserAgentTable"`
	}

	//MetaTableCfg contains the meta db collection names
	MetaTableCfg struct {
		FilesTable     string `yaml:"FilesTable"`
		DatabasesTable string `yaml:"DatabasesTable"`
	}
)

// loadTableConfig attempts to parse a config file
func loadTableConfig(cfgPath string) (*TableCfg, error) {
	var config = new(TableCfg)
	_, err := os.Stat(cfgPath)

	if os.IsNotExist(err) {
		return config, err
	}

	cfgFile, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(cfgFile, config)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read config: %s\n", err.Error())
		return config, err
	}

	// expand env variables, config is a pointer
	// so we have to call elem on the reflect value
	expandConfig(reflect.ValueOf(config).Elem())

	return config, nil
}
