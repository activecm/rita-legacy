package parsetypes

import (
	"github.com/globalsign/mgo/bson"
)

type ExplodedDNS struct {
	// ID contains the id set by mongodb
	ID bson.ObjectId `bson:"_id,omitempty"`
	// Domain
	Domain string `bson:"domain"`
	// Number of subdomains
	Subdomains int64 `bson:"subdomains"`
	// Number of times visited
	Visited int64 `bson:"visited"`
}

//TargetCollection returns the mongo collection this entry should be inserted
//into
// func (in *explodedDNSTable) TargetCollection(config *config.StructureTableCfg) string {
// 	return config.ExplodedDNSTable
// }

// //Indices gives MongoDB indices that should be used with the collection
// func (in *ExplodedDNS) Indices() []string {
// 	return []string{"$hashed:domain"}
// }
