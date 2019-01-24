package useragent

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

	coll := session.DB(targetDB).C("uconn")

	indexes := []mgo.Index{
		{Key: []string{"user_agent"}, Unique: true},
		{Key: []string{"times_used"}},
	}

	for _, index := range indexes {
		err := coll.EnsureIndex(index)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *repo) Upsert(userAgent *parsetypes.UserAgent, targetDB string) error {
	r.db.SelectDB(targetDB)
	session := r.db.Session.Copy()
	defer session.Close()

	coll := session.DB(targetDB).C("useragent")

	// set up update query
	query := bson.D{
		{"$setOnInsert", bson.M{"user_agent": userAgent.UserAgent}},
		{"$inc", bson.M{"times_used": 1}},
	}

	selector := bson.M{"user_agent": userAgent.UserAgent}
	// update or insert to useragent collection
	_, err := coll.Upsert(
		selector,
		query)

	if err != nil {
		return err
	}
	return nil
}
