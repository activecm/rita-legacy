package explodeddns

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

//CreateIndexes ....
func (r *repo) CreateIndexes() error {
	session := r.res.DB.Session.Copy()
	defer session.Close()

	coll := session.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.DNS.ExplodedDNSTable)

	indexes := []mgo.Index{
		{Key: []string{"domain"}, Unique: true},
		{Key: []string{"visited"}},
		{Key: []string{"subdomains"}},
	}

	for _, index := range indexes {
		err := coll.EnsureIndex(index)
		if err != nil {
			return err
		}
	}
	return nil
}

//Upsert loops through every domain ....
func (r *repo) Upsert(domainMap map[string]int) {

	//Create the workers
	writerWorker := newWriter(r.res.Config.T.DNS.ExplodedDNSTable, r.res.DB, r.res.Config)

	dnsAnalyzerWorker := newAnalyzer(
		writerWorker.collect,
		writerWorker.close,
	)

	//kick off the threaded goroutines
	for i := 0; i < 1; i++ { //util.Max(1, runtime.NumCPU()/2)
		dnsAnalyzerWorker.start()
		writerWorker.start()
	}

	for entry, count := range domainMap {

		dnsAnalyzerWorker.collect(domain{entry, count})

	}
}
