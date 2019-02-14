package beacon

import (
	"runtime"

	"github.com/activecm/rita/pkg/uconn"
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

	collectionName := r.res.Config.T.Beacon.BeaconTable

	// Desired indexes
	indexes := []mgo.Index{
		{Key: []string{"-score"}},
		{Key: []string{"$hashed:src"}},
		{Key: []string{"$hashed:dst"}},
		{Key: []string{"-connection_count"}},
	}
	err := r.res.DB.CreateCollection(collectionName, indexes)
	if err != nil {
		return err
	}

	return nil
}

//Upsert loops through every new uconn ....
func (r *repo) Upsert(uconnMap map[string]*uconn.Pair) {
	//Create the workers
	writerWorker := newWriter(r.res.Config.T.Beacon.BeaconTable, r.res.DB, r.res.Config)

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

	for _, entry := range uconnMap {

		analyzerWorker.collect(entry)

	}
}
