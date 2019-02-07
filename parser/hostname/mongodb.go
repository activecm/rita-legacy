package hostname

import (
	"runtime"

	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
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

	// set collection name
	collectionName := r.res.Config.T.DNS.HostnamesTable

	// check if collection already exists
	names, _ := session.DB(r.res.DB.GetSelectedDB()).CollectionNames()

	// if collection exists, we don't need to do anything else
	for _, name := range names {
		if name == collectionName {
			return nil
		}
	}

	// set desired indexes
	indexes := []mgo.Index{{Key: []string{"host"}, Unique: true}}

	// create collection
	err := r.res.DB.CreateCollection(collectionName, indexes)
	if err != nil {
		return err
	}

	return nil
}

//Upsert loops through every domain ....
func (r *repo) Upsert(hostnameMap map[string][]string) {

	//Create the workers
	writerWorker := newWriter(r.res.Config.T.DNS.HostnamesTable, r.res.DB, r.res.Config)

	hostnameAnalyzerWorker := newAnalyzer(
		r.res.DB,
		r.res.Config,
		writerWorker.collect,
		writerWorker.close,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		hostnameAnalyzerWorker.start()
		writerWorker.start()
	}

	for entry, answers := range hostnameMap {
		hostnameAnalyzerWorker.collect(hostname{entry, answers})

	}
}
