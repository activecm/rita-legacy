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

//AnalysisView used in order to join the uconn and beacon tables
type AnalysisView struct {
	Src         string  `bson:"src"`
	Dst         string  `bson:"dst"`
	Connections int64   `bson:"connection_count"`
	AvgBytes    float64 `bson:"avg_bytes"`
	Ts          TSData  `bson:"ts"`
	Ds          DSData  `bson:"ds"`
	Score       float64 `bson:"score"`
}

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
