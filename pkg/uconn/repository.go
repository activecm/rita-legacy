package uconn

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnMap map[string]*Input)
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

//Input holds aggregated connection information between two hosts in a dataset
type Input struct {
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

//LongConnResult represents a pair of hosts that communicated and
//the longest connection between those hosts.
type LongConnResult struct {
	data.UniqueIPPair `bson:",inline"`
	MaxDuration       float64  `bson:"maxdur"`
	Tuples            []string `bson:"tuples"`
}
