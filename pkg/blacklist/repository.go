package blacklist

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

// Repository for blacklist results in host collection
type Repository interface {
	Upsert()
}

//hostsUpdate is used to update the hosts table with blacklisted source and destinations
type hostsUpdate struct {
	selector bson.M
	query    bson.M
}

//connectionPeer records how many connections were made to/ from a given host and how many bytes were sent/ received
type connectionPeer struct {
	Host        data.UniqueIP `bson:"_id"`
	Connections int           `bson:"bl_conn_count"`
	TotalBytes  int           `bson:"bl_total_bytes"`
}

//ResultsView for blacklisted ips (for reporting)
type ResultsView struct {
	Host              data.UniqueIP   `bson:",inline"`
	Peers             []data.UniqueIP `bson:"peers"`
	Connections       int             `bson:"conn_count"`
	UniqueConnections int             `bson:"uconn_count"`
	TotalBytes        int             `bson:"total_bytes"`
}
