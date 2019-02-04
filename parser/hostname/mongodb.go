package hostname

import (
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo"
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

func (r *repo) CreateIndexes() error {
	session := r.res.DB.Session.Copy()
	defer session.Close()

	coll := session.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.DNS.HostnamesTable)

	indexes := []mgo.Index{{Key: []string{"host"}, Unique: true}}

	for _, index := range indexes {
		err := coll.EnsureIndex(index)
		if err != nil {
			return err
		}
	}
	return nil
}

//Upsert loops through every domain ....
func (r *repo) Upsert(hostnameMap map[string][]string) {

	//Create the workers
	writerWorker := newWriter(r.res.Config.T.DNS.HostnamesTable, r.res.DB, r.res.Config)

	analyzerWorker := newAnalyzer(
		writerWorker.collect,
		writerWorker.close,
	)

	//kick off the threaded goroutines
	for i := 0; i < 1; i++ { //util.Max(1, runtime.NumCPU()/2)
		analyzerWorker.start()
		writerWorker.start()
	}

	for entry, answers := range hostnameMap {

		analyzerWorker.collect(hostname{entry, answers})

	}
}
