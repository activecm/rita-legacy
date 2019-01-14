package uconn

import(
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/database"
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