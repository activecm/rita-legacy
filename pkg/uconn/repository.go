package uconn

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnMap map[string]*Pair)
}

//updateInfo ....
type updateInfo struct {
	selector interface{}
	query    interface{}
}

//update ....
type update struct {
	uconn      updateInfo
	hostMaxDur updateInfo
}

//Pair ....
type Pair struct {
	Src             string
	Dst             string
	ConnectionCount int64
	IsLocalSrc      bool
	IsLocalDst      bool
	TotalBytes      int64
	MaxDuration     float64
	TotalDuration   float64
	TsList          []int64
	OrigBytesList   []int64
	Tuples          []string
	// InvalidCerts    []string
	InvalidCertFlag bool
	UPPSFlag        bool
}

//LongConnAnalysisView (for reporting)
type LongConnAnalysisView struct {
	Src         string   `bson:"src"`
	Dst         string   `bson:"dst"`
	MaxDuration float64  `bson:"maxdur"`
	Tuples      []string `bson:"tuples"`
	TupleStr    string
}
