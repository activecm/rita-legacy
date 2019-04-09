package blacklist

import (
	"fmt"
	"runtime"

	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo/bson"
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

//Upsert loops through every domain ....
func (r *repo) Upsert() {

	session := r.res.DB.Session.Copy()
	defer session.Close()

	iter := session.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.Structure.HostTable).Find(bson.M{"blacklisted": true}).Iter()

	//Create the workers
	writerWorker := newWriter(r.res.Config.T.Structure.HostTable, r.res.DB, r.res.Config)

	analyzerWorker := newAnalyzer(
		r.res.Config.S.Bro.CurrentChunk,
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

	var res hostRes
	fmt.Println("\t[-] Updating blacklisted peers ...")
	// loop over map entries
	for iter.Next(&res) {

		analyzerWorker.collect(res)

	}

	// start the closing cascade (this will also close the other channels)
	analyzerWorker.close()

}
