package host

import (
	"fmt"
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

	coll := session.DB(targetDB).C("host")

	// create hosts collection
	// Desired indexes
	indexes := []mgo.Index{
		{Key: []string{"ip"}, Unique: true},
		{Key: []string{"local"}},
		{Key: []string{"ipv4"}},
		{Key: []string{"ipv4_binary"}},
	}

	for _, index := range indexes {
		err := coll.EnsureIndex(index)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *repo) Upsert(host *parsetypes.Host, isSrc bool, targetDB string) error {
	r.db.SelectDB(targetDB)
	session := r.db.Session.Copy()
	defer session.Close()

	coll := session.DB(targetDB).C("host")

	// set up update query
	query := bson.D{
		{"$setOnInsert", bson.M{"local": host.Local}},
		{"$setOnInsert", bson.M{"ipv4": host.IPv4}},
		{"$max", bson.M{"max_duration": host.MaxDuration}},
		{"$setOnInsert", bson.M{"ipv4_binary": host.IPv4Binary}},
	}

	if isSrc {
		query = append(query, bson.DocElem{"$inc", bson.M{"count_src": 1}})
	} else {
		query = append(query, bson.DocElem{"$inc", bson.M{"count_dst": 1}})
	}

	// update hosts field
	_, err := coll.Upsert(
		bson.M{"ip": host.IP},
		query)

	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
