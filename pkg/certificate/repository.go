package certificate

import (
	"github.com/activecm/rita-legacy/pkg/data"
)

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Upsert(useragentMap map[string]*Input)
}

// Input ....
type Input struct {
	Host         data.UniqueIP
	Seen         int64
	OrigIps      data.UniqueIPSet
	InvalidCerts data.StringSet
	Tuples       data.StringSet
}

// AnalysisView (for reporting)
type AnalysisView struct {
	UserAgent string `bson:"user_agent"`
	TimesUsed int64  `bson:"seen"`
}
