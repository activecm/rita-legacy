package explodeddns

import (
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type repo struct {
	db *database.DB
}

//NewMongoRepository create new repository
func NewMongoRepository(database *database.DB) Repository {
	return &repo{
		db: database,
	}
}

func (r *repo) CreateIndexes(targetDB string) error {
	r.db.SelectDB(targetDB)
	session := r.db.Session.Copy()
	defer session.Close()

	coll := session.DB(targetDB).C("explodedDNS")

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

func (r *repo) Upsert(explodedDNS *parsetypes.ExplodedDNS, targetDB string) error {
	r.db.SelectDB(targetDB)
	session := r.db.Session.Copy()
	defer session.Close()

	coll := session.DB(targetDB).C("explodedDNS")

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
