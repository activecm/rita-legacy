package beacon

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/globalsign/mgo"
)

// Repository for beacon collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnMap map[string]*uconn.Input, hostMap map[string]*host.Input, minTimestamp, maxTimestamp int64)
}

type mgoBulkAction func(*mgo.Bulk) int

type mgoBulkActions map[string]mgoBulkAction

//TSData ...
type TSData struct {
	Range      int64   `bson:"range"`
	Mode       int64   `bson:"mode"`
	ModeCount  int64   `bson:"mode_count"`
	Skew       float64 `bson:"skew"`
	Dispersion int64   `bson:"dispersion"`
	Duration   float64 `bson:"duration"`
}

//DSData ...
type DSData struct {
	Skew       float64 `bson:"skew"`
	Dispersion int64   `bson:"dispersion"`
	Range      int64   `bson:"range"`
	Mode       int64   `bson:"mode"`
	ModeCount  int64   `bson:"mode_count"`
}

//Result represents a beacon between two hosts. Contains information
//on connection delta times and the amount of data transferred
type Result struct {
	data.UniqueIPPair `bson:",inline"`
	Connections       int64   `bson:"connection_count"`
	AvgBytes          float64 `bson:"avg_bytes"`
	TotalBytes        int64   `bson:"total_bytes"`
	Ts                TSData  `bson:"ts"`
	Ds                DSData  `bson:"ds"`
	Score             float64 `bson:"score"`
}

//StrobeResult represents a unique connection with a large amount
//of connections between the hosts
type StrobeResult struct {
	data.UniqueIPPair `bson:",inline"`
	ConnectionCount   int64 `bson:"connection_count"`
}
