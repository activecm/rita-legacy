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

// upsertInfo captures the parameters needed to call mgo .Update or .Upsert against a collection
type upsertInfo struct {
	selector bson.M
	query    bson.M
}

// updateWithArrayFiltersInfo captures the parameters needed to call mgo .UpdateWithArrayFilters against a collection
type updateWithArrayFiltersInfo struct {
	selector     bson.M
	query        bson.M
	arrayFilters []bson.M
}

// update represents MongoDB updates to be carried out by the writer
type update struct {
	useragent upsertInfo
	host      updateWithArrayFiltersInfo
}

//Input ....
type Input struct {
	Name     string
	Seen     int64
	OrigIps  data.UniqueIPSet
	Requests []string
	JA3      bool
}

//Result represents a user agent and how many times that user agent
//was seen in the dataset
type Result struct {
	UserAgent string `bson:"user_agent"`
	TimesUsed int64  `bson:"seen"`
}
