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
	OpenBytes       int64
	TotalBytes      int64
	MaxDuration     float64
	OpenDuration    float64
	TotalDuration   float64
	TsList          []int64
	OrigBytesList   []int64
	Tuples          []string
	// InvalidCerts    []string
	InvalidCertFlag bool
	UPPSFlag        bool
	ConnStateList   map[string]*ConnState
}

//LongConnResult represents a pair of hosts that communicated and
//the longest connection between those hosts.
type LongConnResult struct {
	data.UniqueIPPair `bson:",inline"`
	MaxDuration       float64  `bson:"maxdur"`
	Tuples            []string `bson:"tuples"`
	Open              bool     `bson:"open"`
}

//ConnState is used to determine if a particular
// connection, keyed by zeek's UID field, is open
// or closed. If a connection is still open, we
// will write its bytes and duration info out in
// a separate field in mongo. This is needed so
// that we can keep a running score of data from
// open connections without double-counting the information
// when the connection closes.
// Parameters:
//		Bytes: 		total bytes for current connection
//		DstPort: 	destination port of current connection	(not currently used)
//		Duration: 	total duration for current connection
//		Open:		shows if a connection is still open
type ConnState struct {
	Bytes    int64   `bson:"bytes"`
	DstPort  int     `bson:"dstPort"`
	Duration float64 `bson:"duration"`
	Open     bool    `bson:"open"`
}
