package beacon

import "github.com/activecm/rita/parser/uconn"

// Repository for host collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnMap map[string]uconn.Pair)
}

//update ....
type update struct {
	beaconSelector interface{}
	beaconQuery    interface{}
	hostSelector   interface{}
	hostQuery      interface{}
}

type uconnRes struct {
	Src             string  `bson:"src"`
	Dst             string  `bson:"dst"`
	TsList          []int64 `bson:"ts_list"`
	OrigIPBytes     []int64 `bson:"orig_bytes_list"`
	ConnectionCount int     `bson:"connection_count"`
	AverageBytes    float32 `bson:"avg_bytes"`
}
