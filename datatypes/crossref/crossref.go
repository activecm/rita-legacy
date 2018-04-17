package crossref

import (
	"github.com/activecm/rita/database"
)

type (
	XRef struct {
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
