package certificate

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Upsert(useragentMap map[string]*Input)
}

//update ....
type update struct {
	selector   bson.M
	query      bson.M
	collection string
}

//Input ....
type Input struct {
	Host         data.UniqueIP
	Seen         int64
	OrigIps      data.UniqueIPSet
	InvalidCerts data.StringSet
	Tuples       data.StringSet
}

//AnalysisView (for reporting)
type AnalysisView struct {
	UserAgent string `bson:"user_agent"`
	TimesUsed int64  `bson:"seen"`
}
