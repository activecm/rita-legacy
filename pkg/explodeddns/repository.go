package explodeddns

import "github.com/globalsign/mgo/bson"

// Repository for explodedDNS collection
type Repository interface {
	CreateIndexes() error
	// Upsert(explodedDNS *parsetypes.ExplodedDNS) error
	Upsert(domainMap map[string]int)
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
	newExplodedDNS      upsertInfo
	existingExplodedDNS updateWithArrayFiltersInfo
}

//domain ....
type domain struct {
	name  string
	count int
}

//dns ....
type dns struct {
	Domain         string `bson:"domain"`
	SubdomainCount int64  `bson:"subdomain_count"`
	CID            int    `bson:"cid"`
}

//Result represents a hostname, how many subdomains were found
//for that hostname, and how many times that hostname and its subdomains
//were looked up.
type Result struct {
	Domain         string `bson:"domain"`
	SubdomainCount int64  `bson:"subdomain_count"`
	Visited        int64  `bson:"visited"`
}
