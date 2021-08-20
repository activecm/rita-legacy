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
	Hosts               data.UniqueIPPair
	OpenConnectionCount int64
	ConnectionCount     int64
	IsLocalSrc          bool
	IsLocalDst          bool
	OpenBytes           int64
	TotalBytes          int64
	MaxDuration         float64
	OpenDuration        float64
	TotalDuration       float64
	OpenTSList          []int64
	TsList              []int64
	OrigBytesList       []int64
	OpenOrigBytes       int64
	Tuples              data.StringSet
	InvalidCertFlag     bool
	UPPSFlag            bool
	ConnStateMap        map[string]*ConnState
}

//LongConnResult represents a pair of hosts that communicated and
//the longest connection between those hosts.
type LongConnResult struct {
	data.UniqueIPPair `bson:",inline"`
	MaxDuration       float64  `bson:"maxdur"`
	Tuples            []string `bson:"tuples"`
	Open              bool     `bson:"open"`
}

//OpenConnResult represents a pair of hosts that currently
// have an open connection. It shows the current number of
// bytes that have been transferred, the total duration thus far,
// the port:protocol:service tuple, and the Zeek UID in case
// the user wants to look for that connection in their zeek logs
type OpenConnResult struct {
	data.UniqueIPPair `bson:",inline"`
	Bytes             int     `bson:"bytes"`
	Duration          float64 `bson:"duration"`
	Tuple             string  `bson:"tuple"`
	UID               string  `bson:"uid"`
}

//ConnState is used to determine if a particular
// connection, keyed by Zeek's UID field, is open
// or closed. If a connection is still open, we
// will write its bytes and duration info out in
// a separate field in mongo. This is needed so
// that we can keep a running score of data from
// open connections without double-counting the information
// when the connection closes.
// Parameters:
//		Bytes: 		total bytes for current connection
//		Duration: 	total duration for current connection
//		Open:		shows if a connection is still open
//		OrigBytes:  total origin bytes for current connection
//		Ts:			timestamp of the start of the open connection
//		Tuple: 		destination port:protocol:service of current connection
type ConnState struct {
	Bytes     int64   `bson:"bytes"`
	Duration  float64 `bson:"duration"`
	Open      bool    `bson:"open"`
	OrigBytes int64   `bson:"orig_bytes"`
	Ts        int64   `bson:"ts"`
	Tuple     string  `bson:"tuple"`
}
