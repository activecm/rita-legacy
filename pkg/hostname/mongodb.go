package hostname

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

	coll := session.DB(targetDB).C("hostnames")

	indexes := []mgo.Index{{Key: []string{"host"}, Unique: true}}

	for _, index := range indexes {
		err := coll.EnsureIndex(index)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *repo) Upsert(hostname *parsetypes.Hostname, targetDB string) error {
	r.db.SelectDB(targetDB)
	session := r.db.Session.Copy()
	defer session.Close()

	coll := session.DB(targetDB).C("hostnames")

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
