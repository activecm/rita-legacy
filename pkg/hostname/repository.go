package hostname

// Repository for hostnames collection
type Repository interface {
	CreateIndexes() error
	// Upsert(hostname *parsetypes.Hostname) error
	Upsert(domainMap map[string][]string)
}

//update ....
type update struct {
	selector interface{}
	query    interface{}
}

type hostname struct {
	host string   `bson:"host"`
	ips  []string `bson:"ips"`
}

//ritaBLResult contains the summary of a result from the "ip" collection of rita-bl
type ritaBLResult struct {
	index string `bson:"index"` // Potentially malicious IP
	list  string `bson:"list"`  // which blacklist ip was listed on
}

//uconnRes (mystery: won't work if you change to lowercase, even though not exported ????)
type uconnRes struct {
	Connections       int `bson:"conn_count"`
	UniqueConnections int `bson:"uconn_count"`
	TotalBytes        int `bson:"total_bytes"`
}

//AnalysisView (for reporting)
type AnalysisView struct {
	Host              string   `bson:"host"`
	IPs               []string `bson:"ips"`
	Connections       int      `bson:"conn_count"`
	UniqueConnections int      `bson:"uconn_count"`
	TotalBytes        int      `bson:"total_bytes"`
	ConnectedHosts    []string `bson:",omitempty"`
}
