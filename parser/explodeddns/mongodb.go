package explodeddns

import (
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type repo struct {
	res *resources.Resources
}

//NewMongoRepository create new repository
func NewMongoRepository(res *resources.Resources) Repository {
	return &repo{
		res: res,
	}
}

func (r *repo) CreateIndexes() error {
	session := r.res.DB.Session.Copy()
	defer session.Close()

	coll := session.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.DNS.ExplodedDNSTable)

	indexes := []mgo.Index{
		{Key: []string{"domain"}, Unique: true},
		//{Key: []string{"subdomains"}},
	}

	for _, index := range indexes {
		err := coll.EnsureIndex(index)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *repo) Upsert(explodedDNS *parsetypes.ExplodedDNS) error {
	session := r.res.DB.Session.Copy()
	defer session.Close()

	coll := session.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.DNS.ExplodedDNSTable)

	// set up update query
	query := bson.D{
		{"$setOnInsert", bson.M{"domain": explodedDNS.Domain}},
		//{"$setOnInsert", bson.M{"subdomains": explodedDNS.Subdomains}},
		//{"$inc", bson.M{"visited": explodedDNS.Visited}},
	}

	selector := bson.M{"domain": explodedDNS.Domain}
	// update hosts field
	_, err := coll.Upsert(
		selector,
		query)

	if err != nil {
		return err
	}
	return nil
}
