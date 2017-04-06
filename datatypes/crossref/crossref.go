package crossref

import (
	"github.com/bglebrun/rita/database"
	"gopkg.in/mgo.v2/bson"
)

type (
	XRef struct {
		ID         bson.ObjectId `bson:"_id,omitempty"`
		ModuleName string        `bson:"module"`
		Host       string        `bson:"host"`
	}

	//XRefSelector selects internal and external hosts from analysis modules
	XRefSelector interface {
		// GetName returns the name of the analyis module
		GetName() string
		// Select returns channels containgin the internal and external hosts
		Select(*database.Resources) (<-chan string, <-chan string)
	}
)
