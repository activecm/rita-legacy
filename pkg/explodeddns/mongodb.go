package explodeddns

import (
	"runtime"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"

	log "github.com/sirupsen/logrus"
)

type repo struct {
	database *database.DB
	config   *config.Config
	log      *log.Logger
}

// NewMongoRepository bundles the given resources for updating MongoDB with exploded DNS data
func NewMongoRepository(db *database.DB, conf *config.Config, logger *log.Logger) Repository {
	return &repo{
		database: db,
		config:   conf,
		log:      logger,
	}
}

// CreateIndexes creates indexes for the explodedDns collection
func (r *repo) CreateIndexes() error {
	session := r.database.Session.Copy()
	defer session.Close()

	// set collection name
	collectionName := r.config.T.DNS.ExplodedDNSTable

	// check if collection already exists
	names, _ := session.DB(r.database.GetSelectedDB()).CollectionNames()

	// if collection exists, we don't need to do anything else
	for _, name := range names {
		if name == collectionName {
			return nil
		}
	}

	// set desired indexes
	indexes := []mgo.Index{
		{Key: []string{"domain"}, Unique: true},
		// {Key: []string{"visited"}},
		{Key: []string{"subdomain_count"}},
	}

	// create collection
	err := r.database.CreateCollection(collectionName, indexes)
	if err != nil {
		return err
	}

	return nil
}

// Upsert records the given dns query count data in MongoDB
func (r *repo) Upsert(domainMap map[string]int) {

	//Create the workers
	writerWorker := database.NewBulkWriter(r.database, r.config, r.log, true, "exploded_dns")

	analyzerWorker := newAnalyzer(
		r.config.S.Rolling.CurrentChunk,
		r.database,
		r.config,
		writerWorker.Collect,
		writerWorker.Close,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		analyzerWorker.start()
		writerWorker.Start()
	}

	// progress bar for troubleshooting
	p := mpb.New(mpb.WithWidth(20))
	bar := p.AddBar(int64(len(domainMap)),
		mpb.PrependDecorators(
			decor.Name("\t[-] Exploded DNS Analysis:", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over map entries
	for entry, count := range domainMap {
		//Mongo Index key is limited to a size of 1024 https://docs.mongodb.com/v3.4/reference/limits/#index-limitations
		//  so if the key is too large, we should cut it back, this is rough but
		//  works. Figured 800 allows some wiggle room, while also not being too large
		if len(entry) > 1024 {
			entry = entry[:800]
		}
		analyzerWorker.collect(domain{entry, count})
		bar.IncrBy(1)
	}

	p.Wait()

	// start the closing cascade (this will also close the other channels)
	analyzerWorker.close()
}
