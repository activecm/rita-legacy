package host

import "github.com/activecm/rita/pkg/uconn"

// Repository for host collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnMap map[string]*uconn.Pair)
}

//update ....
type update struct {
	selector interface{}
	query    interface{}
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

//AnalysisView for blacklisted ips (for reporting)
type AnalysisView struct {
	Host              string   `bson:"host"`
	Connections       int      `bson:"conn_count"`
	UniqueConnections int      `bson:"uconn_count"`
	TotalBytes        int      `bson:"total_bytes"`
	ConnectedHosts    []string `bson:"ips,omitempty"`
}
