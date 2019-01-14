package freq

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

func (r *repo) Insert(freq *parsetypes.Freq, targetDB string) error {
	session := r.pool.Session(nil)
	defer session.Close()
	coll := session.DB(targetDB).C("freq")

	err := coll.Insert(freq)

	if err != nil {
		return err
	}
	return nil
}