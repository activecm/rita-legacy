package blacklist

//BlacklistedIP holds information on a blacklisted IP address and
//the summary statistics on the host
type BlacklistedIP struct {
	IP                string   `bson:"ip"`
	Connections       int      `bson:"conn"`
	UniqueConnections int      `bson:"uconn"`
	TotalBytes        int      `bson:"total_bytes"`
	Lists             []string `bson:"lists"`
	ConnectedHosts    []string `bson:",omitempty"`
}

//BlacklistedHostname holds information on a blacklisted hostname and
//the summary statistics associated with the hosts behind the hostname
type BlacklistedHostname struct {
	Hostname          string   `bson:"hostname"`
	Connections       int      `bson:"conn"`
	UniqueConnections int      `bson:"uconn"`
	TotalBytes        int      `bson:"total_bytes"`
	Lists             []string `bson:"lists"`
	ConnectedHosts    []string `bson:",omitempty"`
}
