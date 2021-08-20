package blacklist

import (
	"fmt"
	"runtime"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/util"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	log "github.com/sirupsen/logrus"
)

type repo struct {
	database *database.DB
	config   *config.Config
	log      *log.Logger
}

//NewMongoRepository create new repository
func NewMongoRepository(db *database.DB, conf *config.Config, logger *log.Logger) Repository {
	return &repo{
		database: db,
		config:   conf,
		log:      logger,
	}
}

//CreateIndexes sets up the indices needed to find hosts which contacted blacklisted hosts
func (r *repo) CreateIndexes() error {
	session := r.database.Session.Copy()
	defer session.Close()

	coll := session.DB(r.database.GetSelectedDB()).C(r.config.T.Structure.HostTable)

	// create hosts collection
	// Desired indexes
	indexes := []mgo.Index{
		{Key: []string{"dat.bl.ip", "dat.bl.network_uuid"}},
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
func (r *repo) Upsert() {

	session := r.database.Session.Copy()
	defer session.Close()

	iter := session.DB(r.database.GetSelectedDB()).C(r.config.T.Structure.HostTable).Find(bson.M{"blacklisted": true}).Iter()

	//Create the workers
	writerWorker := newWriter(r.config.T.Structure.HostTable, r.database, r.config, r.log)

	analyzerWorker := newAnalyzer(
		r.config.S.Rolling.CurrentChunk,
		r.database,
		r.config,
		writerWorker.collect,
		writerWorker.close,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		analyzerWorker.start()
		writerWorker.start()
	}

	var res data.UniqueIP
	fmt.Println("\t[-] Updating blacklisted peers ...")
	// loop over map entries
	for iter.Next(&res) {

		analyzerWorker.collect(res)

	}

	// start the closing cascade (this will also close the other channels)
	analyzerWorker.close()

}
