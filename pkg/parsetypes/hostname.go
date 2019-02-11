package parsetypes

import (
	"github.com/globalsign/mgo/bson"
)

type (
	// Hostname collection
	Hostname struct {
		ID   bson.ObjectId `bson:"_id,omitempty"`
		Host string        `bson:"host"`
		IPs  []string      `bson:"ips"`
	}
)
