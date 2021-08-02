package blacklist

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

// Repository for blacklist results in host collection
type Repository interface {
	CreateIndexes() error
	Upsert()
}

// updateWithArrayFiltersInfo captures the parameters needed to call mgo .UpdateWithArrayFilters against a collection
type updateWithArrayFiltersInfo struct {
	selector     bson.M
	query        bson.M
	arrayFilters []bson.M
}

//connectionPeer records how many connections were made to/ from a given host and how many bytes were sent/ received.
//used to read data from MongoBD in getUniqueConnsforBLSource() and getUniqueConnsforBLDestination()
type connectionPeer struct {
	Host        data.UniqueIP `bson:"_id"`
	Connections int           `bson:"bl_conn_count"`
	TotalBytes  int           `bson:"bl_total_bytes"`
}

//IPResult represtes a blacklisted IP and summary data
//about the connections involving that IP
type IPResult struct {
	Host              data.UniqueIP   `bson:",inline"`
	Connections       int             `bson:"conn_count"`
	UniqueConnections int             `bson:"uconn_count"`
	TotalBytes        int             `bson:"total_bytes"`
	Peers             []data.UniqueIP `bson:"peers"`
}

//HostnameResult represents a blacklisted hostname and summary
//data about the connections made to that hostname
type HostnameResult struct {
	Host              string          `bson:"host"`
	Connections       int             `bson:"conn_count"`
	UniqueConnections int             `bson:"uconn_count"`
	TotalBytes        int             `bson:"total_bytes"`
	ConnectedHosts    []data.UniqueIP `bson:"sources,omitempty"`
}
