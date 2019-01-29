package hostname

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

	coll := session.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.DNS.HostnamesTable)

	indexes := []mgo.Index{{Key: []string{"host"}, Unique: true}}

	for _, index := range indexes {
		err := coll.EnsureIndex(index)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *repo) Upsert(hostname *parsetypes.Hostname) error {
	session := r.res.DB.Session.Copy()
	defer session.Close()

	coll := session.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.DNS.HostnamesTable)

	// set up update query
	query := bson.D{
		{"$setOnInsert", bson.M{"host": hostname.Host}},
		{"$addToSet", bson.M{"ips": bson.M{"$each": hostname.IPs}}},
	}

	selector := bson.M{"host": hostname.Host}
	// update hostnames collection
	_, err := coll.Upsert(
		selector,
		query)

	if err != nil {
		return err
	}
	return nil
}
