package blacklist

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

// Repository for blacklist results in host collection
type Repository interface {
	Upsert()
}

//hostsUpdate is used to update the hosts table with blacklisted source and destinations
type hostsUpdate struct {
	selector bson.M
	query    bson.M
}

//connectionPeer records how many connections were made to/ from a given host and how many bytes were sent/ received
type connectionPeer struct {
	Host        data.UniqueIP `bson:"_id"`
	Connections int           `bson:"bl_conn_count"`
	TotalBytes  int           `bson:"bl_total_bytes"`
}

//IP ....
// type IP struct {
// 	Host                  string
// 	IsLocal               bool
// 	CountSrc              int
// 	CountDst              int
// 	ConnectionCount       int64
// 	TotalBytes            int64
// 	MaxDuration           float64
// 	TotalDuration         float64
// 	TXTQueryCount         int64
// 	UntrustedAppConnCount int64
// 	MaxTS                 int64
// 	MinTS                 int64
// 	ConnectedSrcHosts     []string
// 	ConnectedDstHosts     []string
// 	IP4                   bool
// 	IP4Bin                int64
// }
//
// //AnalysisView for blacklisted ips (for reporting)
// type AnalysisView struct {
// 	Host              string   `bson:"host"`
// 	Connections       int      `bson:"conn_count"`
// 	UniqueConnections int      `bson:"uconn_count"`
// 	TotalBytes        int      `bson:"total_bytes"`
// 	ConnectedHosts    []string `bson:"ips,omitempty"`
// }
