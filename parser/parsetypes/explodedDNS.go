package parsetypes

import (
	"github.com/globalsign/mgo/bson"
)

// ExplodedDNS contains domain info
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
