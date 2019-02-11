package uconn

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnMap map[string]Pair)
}

//update ....
type update struct {
	selector interface{}
	query    interface{}
}

//Pair ....
type Pair struct {
	Src             string
	Dst             string
	ConnectionCount int64
	IsLocalSrc      bool
	IsLocalDst      bool
	TotalBytes      int64
	AvgBytes        float64
	MaxDuration     float64
	TotalDuration   float64
	TsList          []int64
	OrigBytesList   []int64
}
