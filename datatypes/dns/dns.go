package dns

import "gopkg.in/mgo.v2/bson"

type (
	ExplodedDNS struct {
		ID         bson.ObjectId `bson:"_id,omitempty"`
		Domain     string        `bson:"domain"`
		Subdomains int64         `bson:"subdomains"`
		Visited    int64         `bson:"visited"`
	}
)
