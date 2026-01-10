package host

import (
	"github.com/activecm/rita-legacy/pkg/data"
)

// Repository for host collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnMap map[string]*Input)
}

// Input ...
type Input struct {
	Host                  data.UniqueIP
	IsLocal               bool
	CountSrc              int
	CountDst              int
	ConnectionCount       int64
	TotalBytes            int64
	MaxDuration           float64
	TotalDuration         float64
	UntrustedAppConnCount int64
	MaxTS                 int64
	MinTS                 int64
	IP4                   bool
	IP4Bin                int64
}
