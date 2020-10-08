package host

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

// Repository for host collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnMap map[string]*Input)
}

//update ....
type update struct {
	selector bson.M
	query    bson.M
}

//Input ...
type Input struct {
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

//TODO: Remove
//AnalysisView for blacklisted ips (for reporting)
type AnalysisView struct {
	Host              string   `bson:"host"`
	Connections       int      `bson:"conn_count"`
	UniqueConnections int      `bson:"uconn_count"`
	TotalBytes        int      `bson:"total_bytes"`
	ConnectedHosts    []string `bson:"ips,omitempty"`
}
