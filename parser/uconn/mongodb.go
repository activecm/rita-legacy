package uconn

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
		{Key: []string{"src", "dst"}, Unique: true},
		{Key: []string{"$hashed:src"}},
		{Key: []string{"$hashed:dst"}},
		{Key: []string{"connection_count"}},
	}

	for _, index := range indexes {
		err := coll.EnsureIndex(index)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *repo) Insert(uconn *parsetypes.Uconn, targetDB string) error {
	r.db.SelectDB(targetDB)
	session := r.db.Session.Copy()
	defer session.Close()

	coll := session.DB(targetDB).C("uconn")

	err := coll.Insert(uconn)
	if err != nil {
		return err
	}
	return nil
}

func (r *repo) Upsert(uconn *parsetypes.Uconn, targetDB string) error {
	r.db.SelectDB(targetDB)
	session := r.db.Session.Copy()
	defer session.Close()

	coll := session.DB(targetDB).C("uconn")

	// set up update query
	query := bson.D{
		{"$setOnInsert", bson.M{"src": uconn.Source}},
		{"$setOnInsert", bson.M{"dst": uconn.Destination}},
		{"$setOnInsert", bson.M{"connection_count": uconn.ConnectionCount}},
		{"$setOnInsert", bson.M{"local_src": uconn.LocalSource}},
		{"$setOnInsert", bson.M{"local_dst": uconn.LocalDestination}},
		{"$setOnInsert", bson.M{"total_bytes": uconn.TotalBytes}},
		{"$setOnInsert", bson.M{"avg_bytes": uconn.AverageBytes}},
		{"$setOnInsert", bson.M{"ts_list": uconn.TSList}},
		{"$setOnInsert", bson.M{"orig_bytes_list": uconn.OrigBytesList}},
		{"$setOnInsert", bson.M{"total_duration": uconn.TotalDuration}},
		{"$setOnInsert", bson.M{"max_duration": uconn.MaxDuration}},
	}

	selector := bson.M{"src": uconn.Source, "dst": uconn.Destination}
	// update hosts field
	_, err := coll.Upsert(
		selector,
		query)

	if err != nil {
		return err
	}
	return nil
}
