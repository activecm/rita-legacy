package dns

import "gopkg.in/mgo.v2/bson"

type (
	ExplodedDNS struct {
		ID         bson.ObjectId `bson:"_id,omitempty"`
		Domain     string        `bson:"domain"`
		Subdomains float64       `bson:"subdomains"`
		Visited    float64       `bson:"visited"`
	}
)
