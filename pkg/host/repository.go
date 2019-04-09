package host

// Repository for host collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnMap map[string]*IP)
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

//IP ....
type IP struct {
	Host                  string
	IsLocal               bool
	CountSrc              int
	CountDst              int
	ConnectionCount       int64
	TotalBytes            int64
	MaxDuration           float64
	TotalDuration         float64
	TXTQueryCount         int64
	UntrustedAppConnCount int64
	MaxTS                 int64
	MinTS                 int64
	ConnectedSrcHosts     []string
	ConnectedDstHosts     []string
	IP4                   bool
	IP4Bin                int64
}

//AnalysisView for blacklisted ips (for reporting)
type AnalysisView struct {
	Host              string   `bson:"host"`
	Connections       int      `bson:"conn_count"`
	UniqueConnections int      `bson:"uconn_count"`
	TotalBytes        int      `bson:"total_bytes"`
	ConnectedHosts    []string `bson:"ips,omitempty"`
}
