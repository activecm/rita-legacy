package uconnproxy

import (
	"runtime"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo"
	log "github.com/sirupsen/logrus"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

type repo struct {
	database *database.DB
	config   *config.Config
	log      *log.Logger
}

// NewMongoRepository bundles the given resources for updating MongoDB with proxy connection data
func NewMongoRepository(db *database.DB, conf *config.Config, logger *log.Logger) Repository {
	return &repo{
		database: db,
		config:   conf,
		log:      logger,
	}
}

// CreateIndexes creates indexes for the uconnProxy collection
func (r *repo) CreateIndexes() error {
	session := r.database.Session.Copy()
	defer session.Close()

	// set collection name
	collectionName := r.config.T.Structure.UniqueConnProxyTable

	// check if collection already exists
	names, _ := session.DB(r.database.GetSelectedDB()).CollectionNames()

	// if collection exists, we don't need to do anything else
	for _, name := range names {
		if name == collectionName {
			return nil
		}
	}

	indexes := []mgo.Index{
		{Key: []string{"src", "fqdn", "src_network_uuid"}, Unique: true},
		{Key: []string{"fqdn"}},
		{Key: []string{"src", "src_network_uuid"}},
		{Key: []string{"dat.count"}},
	}

	// create collection
	err := r.database.CreateCollection(collectionName, indexes)
	if err != nil {
		return err
	}

	return nil
}

// Upsert records the given proxy connection data in MongoDB
func (r *repo) Upsert(uconnProxyMap map[string]*Input) {
	// Create the workers
	writerWorker := database.NewBulkWriter(r.database, r.config, r.log, true, "uconnproxy")

	analyzerWorker := newAnalyzer(
		r.config.S.Rolling.CurrentChunk,
		int64(r.config.S.Strobe.ConnectionLimit),
		r.database,
		r.config,
		writerWorker.Collect,
		writerWorker.Close,
	)

	// kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		analyzerWorker.start()
		writerWorker.Start()
	}

	// progress bar for troubleshooting
	p := mpb.New(mpb.WithWidth(20))
	bar := p.AddBar(int64(len(uconnProxyMap)),
		mpb.PrependDecorators(
			decor.Name("\t[-] Uconn Proxy Analysis:", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over map entries
	for _, entry := range uconnProxyMap {
		analyzerWorker.collect(entry)
		bar.IncrBy(1)
	}
	p.Wait()

	// start the closing cascade (this will also close the other channels)
	analyzerWorker.close()
}
