package config

type (
	//TableCfg is the container for other table config sections
	TableCfg struct {
		Log         LogTableCfg
		Blacklisted BlacklistedTableCfg
		DNS         DNSTableCfg
		Crossref    CrossrefTableCfg
		Scanning    ScanningTableCfg
		Structure   StructureTableCfg
		Beacon      BeaconTableCfg
		Urls        UrlsTableCfg
		UserAgent   UserAgentTableCfg
		Meta        MetaTableCfg
	}

	//LogTableCfg contains the configuration for logging
	LogTableCfg struct {
		RitaLogTable string
	}

	//StructureTableCfg contains the names of the base level collections
	StructureTableCfg struct {
		ConnTable       string
		HTTPTable       string
		DNSTable        string
		UniqueConnTable string
		HostTable       string
		IPv4Table       string
		IPv6Table       string
	}

	//BlacklistedTableCfg is used to control the blacklisted analysis module
	BlacklistedTableCfg struct {
		BlacklistDatabase string
		SourceIPsTable    string
		DestIPsTable      string
		HostnamesTable    string
		UrlsTable         string
	}

	//DNSTableCfg is used to control the dns analysis module
	DNSTableCfg struct {
		ExplodedDNSTable string
		HostnamesTable   string
	}

	//CrossrefTableCfg is used to control the crossref analysis module
	CrossrefTableCfg struct {
		SourceTable string
		DestTable   string
	}

	//ScanningTableCfg is used to control the scanning analysis module
	ScanningTableCfg struct {
		ScanTable string
	}

	//BeaconTableCfg is used to control the beaconing analysis module
	BeaconTableCfg struct {
		BeaconTable string
	}

	//UrlsTableCfg is used to control the urls analysis module
	UrlsTableCfg struct {
		UrlsTable string
	}

	//UserAgentTableCfg is used to control the urls analysis module
	UserAgentTableCfg struct {
		UserAgentTable string
	}

	//MetaTableCfg contains the meta db collection names
	MetaTableCfg struct {
		FilesTable     string
		DatabasesTable string
	}
)

// loadTableConfig initializes a config struct
func loadTableConfig() (*TableCfg, error) {
	var config = new(TableCfg)

	// initialize all the table configs
	config.Log.RitaLogTable = "logs"

	config.Structure.ConnTable       = "conn"
	config.Structure.HTTPTable       = "http"
	config.Structure.DNSTable        = "dns"
	config.Structure.UniqueConnTable = "uconn"
	config.Structure.HostTable       = "host"
	config.Structure.IPv4Table       = "ipv4"
	config.Structure.IPv6Table       = "ipv6"

	config.Blacklisted.BlacklistDatabase = "rita-blacklist"
	config.Blacklisted.SourceIPsTable    = "blSourceIPs"
	config.Blacklisted.DestIPsTable      = "blDestIPs"
	config.Blacklisted.HostnamesTable    = "blHostnames"
	config.Blacklisted.UrlsTable         = "blUrls"

	config.DNS.ExplodedDNSTable = "explodedDns"
	config.DNS.HostnamesTable   = "hostnames"

	config.Crossref.SourceTable = "sourceXREF"
	config.Crossref.DestTable   = "destXREF"

	config.Scanning.ScanTable = "scan"

	config.Beacon.BeaconTable = "beacon"

	config.Urls.UrlsTable = "urls"

	config.UserAgent.UserAgentTable = "useragent"

	config.Meta.FilesTable     = "files"
	config.Meta.DatabasesTable = "database"

	return config, nil
}
