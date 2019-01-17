package conn

import (
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/parser/parsetypes"
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

func (r *repo) BulkDelete(conns []*parsetypes.Conn, targetDB string) error {
	r.db.SelectDB(targetDB)
	session := r.db.Session.Copy()
	defer session.Close()

	coll := session.DB(targetDB).C("conn")

	bulk := coll.Bulk()
	bulk.Unordered()

	for _, conn := range conns {
		deleteQuery := bson.M{
			"$and": []bson.M{
				bson.M{"id_orig_h": conn.Source},
				bson.M{"id_resp_h": conn.Destination},
			}}
		bulk.RemoveAll(deleteQuery)
	}

	// Execute the bulk deletion
	_, err := bulk.Run()
	if err != nil {
		return err
	}
	return nil
}
