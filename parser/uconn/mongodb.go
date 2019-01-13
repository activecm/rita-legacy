package uconn

import(
	"github.com/juju/mgosession"
	"github.com/activecm/rita/parser/parsetypes"

	)

type repo struct {
	pool *mgosession.Pool
}

//NewMongoRepository create new repository
func NewMongoRepository(p *mgosession.Pool) Repository {
	return &repo{
		pool: p,
	}
}

func (r *repo) Insert(uconn *parsetypes.Uconn, targetDB string) error {
	session := r.pool.Session(nil)
	coll := session.DB(targetDB).C("uconn")

	err := coll.Insert(uconn)

	if err != nil {
		return err
	}
	return nil
}