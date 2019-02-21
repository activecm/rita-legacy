package config

type (
	//TableCfg is the container for other table config sections
	TableCfg struct {
		Log       LogTableCfg
		DNS       DNSTableCfg
		Structure StructureTableCfg
		Beacon    BeaconTableCfg
		UserAgent UserAgentTableCfg
		Meta      MetaTableCfg
	}

	//LogTableCfg contains the configuration for logging
	LogTableCfg struct {
		RitaLogTable string `default:"logs"`
	}

	//StructureTableCfg contains the names of the base level collections
	StructureTableCfg struct {
		ConnTable       string `default:"conn"`
		HTTPTable       string `default:"http"`
		DNSTable        string `default:"dns"`
		SSLTable        string `default:"ssl"`
		X509Table       string `default:"x509"`
		UniqueConnTable string `default:"uconn"`
		HostTable       string `default:"host"`
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
