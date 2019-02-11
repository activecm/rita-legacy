package useragent

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
	collectionName := r.res.Config.T.UserAgent.UserAgentTable

	// check if collection already exists
	names, _ := session.DB(r.res.DB.GetSelectedDB()).CollectionNames()

	// if collection exists, we don't need to do anything else
	for _, name := range names {
		if name == collectionName {
			return nil
		}
	}

	// set desired indexes
	indexes := []mgo.Index{
		{Key: []string{"user_agent"}, Unique: true},
		{Key: []string{"times_used"}},
	}

	// create collection
	err := r.res.DB.CreateCollection(collectionName, indexes)
	if err != nil {
		return err
	}

	return nil
}

func (r *repo) Upsert(userAgentMap map[string]*Input) {
	//Create the workers
	writerWorker := newWriter(r.res.Config.T.UserAgent.UserAgentTable, r.res.DB, r.res.Config)

	analyzerWorker := newAnalyzer(
		r.res.DB,
		r.res.Config,
		writerWorker.collect,
		writerWorker.close,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		analyzerWorker.start()
		writerWorker.start()
	}

	for key, value := range userAgentMap {
		value.name = key
		analyzerWorker.collect(value)

	}
}
