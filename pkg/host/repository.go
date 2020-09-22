package host

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

// Repository for host collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnMap map[string]*IP)
}

//update ....
type update struct {
	selector bson.M
	query    bson.M
}

//TODO[AGENT]: Convert IP Host to Unique IP. Consider renaming to IP to Input to match other pkgs?
//IP ....
type IP struct {
	Host                  data.UniqueIP
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
