package uconn

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnMap map[string]*Pair)
}

//update ....
type update struct {
	selector interface{}
	query    interface{}
}

//Pair ....
type Pair struct {
	Src                   string
	Dst                   string
	ConnectionCount       int64
	IsLocalSrc            bool
	IsLocalDst            bool
	TotalBytes            int64
	MaxDuration           float64
	TotalDuration         float64
	TsList                []int64
	OrigBytesList         []int64
	TXTQueryCount         int64
	UntrustedAppConnCount int64
}

//AnalysisView (for reporting)
type AnalysisView struct {
	Src             string  `bson:"src"`
	Dst             string  `bson:"dst"`
	LocalSrc        bool    `bson:"local_src"`
	LocalDst        bool    `bson:"local_dst"`
	ConnectionCount int     `bson:"connection_count"`
	TotalBytes      int     `bson:"total_bytes"`
	TsList          []int64 `bson:"ts_list"`         // Connection timestamps for this src, dst pair
	OrigIPBytes     []int64 `bson:"orig_bytes_list"` // Src to dst connection sizes for each connection
	MaxDuration     float32 `bson:"max_duration"`
	TotalDuration   float32 `bson:"total_duration"`
}
