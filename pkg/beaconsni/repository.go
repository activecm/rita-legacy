package beaconsni

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/sniconn"
)

// Repository for beaconsni collection
type Repository interface {
	CreateIndexes() error
	Upsert(tlsMap map[string]*sniconn.TLSInput, httpMap map[string]*sniconn.HTTPInput, hostMap map[string]*host.Input, minTimestamp, maxTimestamp int64)
}

type dissectorResults struct {
	Hosts           data.UniqueSrcFQDNPair
	RespondingIPs   []data.UniqueIP
	ConnectionCount int64
	TotalBytes      int64
	TsList          []int64
	TsListFull      []int64
	OrigBytesList   []int64
}

// Result represents an SNI beacon between a source IP and
// an SNI. An SNI can be comprised of one or more destination IPs.
// Contains information on connection delta times and the amount of data transferred
type Result struct {
	data.UniqueSrcFQDNPair `bson:",inline"`
	Connections            int64   `bson:"connection_count"`
	AvgBytes               float64 `bson:"avg_bytes"`
	TotalBytes             int64   `bson:"total_bytes"`
	Ts                     TSData  `bson:"ts"`
	Ds                     DSData  `bson:"ds"`
	DurScore               float64 `bson:"duration_score"`
	HistScore              float64 `bson:"hist_score"`
	Score                  float64 `bson:"score"`
	// ResolvedIPs            []data.UniqueIP // Requires lookup on SNIconn collection
}

// TSData ...
type TSData struct {
	Score      float64 `bson:"score"`
	Range      int64   `bson:"range"`
	Mode       int64   `bson:"mode"`
	ModeCount  int64   `bson:"mode_count"`
	Skew       float64 `bson:"skew"`
	Dispersion int64   `bson:"dispersion"`
	Duration   float64 `bson:"duration"`
}

// DSData ...
type DSData struct {
	Score      float64 `bson:"score"`
	Skew       float64 `bson:"skew"`
	Dispersion int64   `bson:"dispersion"`
	Range      int64   `bson:"range"`
	Mode       int64   `bson:"mode"`
	ModeCount  int64   `bson:"mode_count"`
}
