package uconn

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnMap map[string]*Pair)
}

//updateInfo ....
type updateInfo struct {
	selector bson.M
	query    bson.M
}

//update ....
type update struct {
	uconn      updateInfo
	hostMaxDur updateInfo
}

//Pair ....
type Pair struct {
	Hosts           data.UniqueIPPair
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
