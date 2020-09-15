package blacklist

// Repository for blacklist results in host collection
type Repository interface {
	Upsert()
}

//update ....
type update struct {
	selector interface{}
	query    interface{}
}

//TODO[AGENT]: Use UniqueIP for Host in blacklist uconnRes
//uconnRes
type uconnRes struct {
	Host              string `bson:"_id"`
	Connections       int    `bson:"bl_conn_count"`
	UniqueConnections int    `bson:"bl_in_count"`
	TotalBytes        int    `bson:"bl_total_bytes"`
}

//TODO[AGENT]: Use UniqueIP for hostres IP in blacklist
type hostRes struct {
	IP string `bson:"ip"`
	// blacklisted bool   `bson:"blacklisted"`
	// CID int `bson:"cid"`
	// dat         []interface{} `bson:"dat"`Host string `bson:"host"`
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
