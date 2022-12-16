package config

type (
	//TableCfg is the container for other table config sections
	TableCfg struct {
		Log         LogTableCfg
		DNS         DNSTableCfg
		Structure   StructureTableCfg
		Beacon      BeaconTableCfg
		BeaconSNI   BeaconSNITableCfg
		BeaconProxy BeaconProxyTableCfg
		UserAgent   UserAgentTableCfg
		Cert        CertificateTableCfg
		Meta        MetaTableCfg
	}

	//LogTableCfg contains the configuration for logging
	LogTableCfg struct {
		RitaLogTable string `default:"logs"`
	}

	//StructureTableCfg contains the names of the base level collections
	StructureTableCfg struct {
		ConnTable            string `default:"conn"`
		DNSTable             string `default:"dns"`
		HostTable            string `default:"host"`
		HTTPTable            string `default:"http"`
		OpenConnTable        string `default:"openconn"`
		SSLTable             string `default:"ssl"`
		UniqueConnTable      string `default:"uconn"`
		UniqueConnProxyTable string `default:"uconnProxy"`
		SNIConnTable         string `default:"SNIconn"`
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

	//BeaconSNITableCfg is used to control the SNI beaconing analysis module
	BeaconSNITableCfg struct {
		BeaconSNITable string `default:"beaconSNI"`
	}

	//BeaconProxyTableCfg is used to control the beaconing analysis module
	BeaconProxyTableCfg struct {
		BeaconProxyTable string `default:"beaconProxy"`
	}

	//UserAgentTableCfg is used to control the useragent analysis module
	UserAgentTableCfg struct {
		UserAgentTable string `default:"useragent"`
	}

	//CertificateTableCfg is used to control the useragent analysis module
	CertificateTableCfg struct {
		CertificateTable string `default:"cert"`
	}

	//MetaTableCfg contains the meta db collection names
	MetaTableCfg struct {
		FilesTable     string `default:"files"`
		DatabasesTable string `default:"databases"`
	}
)
