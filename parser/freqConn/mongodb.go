package freqConn

import(
	"github.com/activecm/rita/parser/parsetypes"
	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
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

func (r *repo) Insert(freqConn *freqConn, targetDB string) error {
	session := r.pool.Session(nil)
	coll := session.DB(targetDB).C("freqConn")

	err := coll.Insert(freqConn)

	if err != nil {
		return err
	}

	return nil
}