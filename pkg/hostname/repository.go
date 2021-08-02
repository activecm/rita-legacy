package hostname

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
)

type (
	// Repository for hostnames collection
	Repository interface {
		CreateIndexes() error
		Upsert(domainMap map[string]*Input)
	}

	//update ....
	update struct {
		selector bson.M
		query    bson.M
	}

	//Input ....
	Input struct {
		Host        string           //A hostname
		ResolvedIPs data.UniqueIPSet //Set of resolved UniqueIPs associated with a given hostname
		ClientIPs   data.UniqueIPSet //Set of DNS Client UniqueIPs which issued queries for a given hostname
	}
)
