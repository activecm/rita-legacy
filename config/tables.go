package config

type (
	//TableCfg is the container for other table config sections
	TableCfg struct {
		Log         LogTableCfg
		Blacklisted BlacklistedTableCfg
		DNS         DNSTableCfg
		Structure   StructureTableCfg
		Beacon      BeaconTableCfg
		Strobe      StrobeTableCfg
		UserAgent   UserAgentTableCfg
		Meta        MetaTableCfg
	}

	//LogTableCfg contains the configuration for logging
	LogTableCfg struct {
		RitaLogTable string `default:"logs"`
	}

	//StructureTableCfg contains the names of the base level collections
	StructureTableCfg struct {
		ConnTable         string `default:"conn"`
		HTTPTable         string `default:"http"`
		DNSTable          string `default:"dns"`
		UniqueConnTable   string `default:"uconn"`
		HostTable         string `default:"host"`
		IPv4Table         string `default:"ipv4"`
		IPv6Table         string `default:"ipv6"`
		FrequentConnTable string `default:"freqConn"`
	}

	//BlacklistedTableCfg is used to control the blacklisted analysis module
	BlacklistedTableCfg struct {
		SourceIPsTable string `default:"blSourceIPs"`
		DestIPsTable   string `default:"blDestIPs"`
		HostnamesTable string `default:"blHostnames"`
	}

	//DNSTableCfg is used to control the dns analysis module
	DNSTableCfg struct {
		ExplodedDNSTable string `default:"explodedDns"`
		HostnamesTable   string `default:"hostnames"`
	}

	//BeaconTableCfg is used to control the beaconing analysis module
	BeaconTableCfg struct {
		BeaconTable string `default:"beacon"`
	}

	//StrobeTableCfg is used to control the strobe analysis module
	StrobeTableCfg struct {
		StrobeTable string `default:"freqConn"`
	}

	//UserAgentTableCfg is used to control the useragent analysis module
	UserAgentTableCfg struct {
		UserAgentTable string `default:"useragent"`
	}

	//MetaTableCfg contains the meta db collection names
	MetaTableCfg struct {
		FilesTable     string `default:"files"`
		DatabasesTable string `default:"databases"`
	}
)