package hostname

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

// Repository for hostnames collection
type Repository interface {
	CreateIndexes() error
	Upsert(domainMap map[string]*Input)
}

//update ....
type update struct {
	selector bson.M
	query    bson.M
}

//Input ....
type Input struct {
	Host        string           //A hostname
	ResolvedIPs data.UniqueIPSet //Set of resolved UniqueIPs associated with a given hostname
	ClientIPs   data.UniqueIPSet //Set of DNS Client UniqueIPs which issued queries for a given hostname
}

//FqdnInput ....
type FqdnInput struct {
	Host            string           //A hostname
	Src             data.UniqueIP    // Single src that connected to a hostname
	ResolvedIPs     data.UniqueIPSet //Set of resolved UniqueIPs associated with a given hostname
	InvalidCertFlag bool
	ConnectionCount int64
	TotalBytes      int64
	TsList          []int64
	OrigBytesList   []int64
}
