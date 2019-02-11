package freq

import (
	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/resources"
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

func (r *repo) Insert(freq *parsetypes.Freq) error {
	session := r.res.DB.Session.Copy()
	defer session.Close()

	coll := session.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.Strobe.StrobeTable)

	err := coll.Insert(freq)

	if err != nil {
		return err
	}
	return nil
}
