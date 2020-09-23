package useragent

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

//TODO[AGENT]: Use UniqueIP with NetworkID for OrigIPs in useragent Input
//Input ....
type Input struct {
	Name     string
	Seen     int64
	OrigIps  data.UniqueIPSet
	Requests []string
	JA3      bool
}

//AnalysisView (for reporting)
type AnalysisView struct {
	UserAgent string `bson:"user_agent"`
	TimesUsed int64  `bson:"seen"`
}
