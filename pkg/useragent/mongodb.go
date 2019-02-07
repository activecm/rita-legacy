package useragent

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

	// set collection name
	collectionName := r.res.Config.T.UserAgent.UserAgentTable

	// check if collection already exists
	names, _ := session.DB(r.res.DB.GetSelectedDB()).CollectionNames()

	// if collection exists, we don't need to do anything else
	for _, name := range names {
		if name == collectionName {
			return nil
		}
	}

	// set desired indexes
	indexes := []mgo.Index{
		{Key: []string{"user_agent"}, Unique: true},
		{Key: []string{"times_used"}},
	}

	// create collection
	err := r.res.DB.CreateCollection(collectionName, indexes)
	if err != nil {
		return err
	}

	return nil
}

func (r *repo) Upsert(userAgent *parsetypes.UserAgent) error {
	session := r.res.DB.Session.Copy()
	defer session.Close()

	coll := session.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.UserAgent.UserAgentTable)

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
