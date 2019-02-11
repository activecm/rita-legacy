package conn

import (
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/resources"
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

func (r *repo) BulkDelete(conns []*parsetypes.Conn) error {
	session := r.res.DB.Session.Copy()
	defer session.Close()

	coll := session.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.Structure.ConnTable)

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
