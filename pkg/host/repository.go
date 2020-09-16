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

//TODO[AGENT]: Convert IP Host to Unique IP. Consider renaming to IP to Input to match other pkgs?
//TODO[AGENT]: Convert ConnectedSrcHosts/ ConnectedDstHosts to []UniqueIP
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
