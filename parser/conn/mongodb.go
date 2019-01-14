package conn

import(
	"github.com/juju/mgosession"
	"github.com/activecm/rita/parser/parsetypes"
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

func (r *repo) BulkDelete(conns []*Conn, targetDB string) error {
	session := r.pool.Session(nil)
	coll := session.DB(targetDB).C("conn")

	bulk := coll.Bulk()
	bulk.Unordered()

	for _, uconn := range uconns {
		deleteQuery := bson.M{
			"$and": []bson.M{
				bson.M{"id_orig_h": uconn.src},
				bson.M{"id_resp_h": uconn.dst},
			}}
		bulk.RemoveAll(deleteQuery)
	}

	// Execute the bulk deletion
	err := bulk.Run()
	if err != nil {
		return err
	}
	return nil
}