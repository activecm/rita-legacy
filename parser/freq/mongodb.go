package freq

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

func (r *repo) Insert(freq *parsetypes.Freq, targetDB string) error {
	r.db.SelectDB(targetDB)
	session := r.db.Session.Copy()
	defer session.Close()

	coll := session.DB(targetDB).C("freq")

	err := coll.Insert(freq)

	if err != nil {
		return err
	}
	return nil
}