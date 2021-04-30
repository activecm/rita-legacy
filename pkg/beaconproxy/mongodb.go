package beaconproxy

import (
	"runtime"
	"time"

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

//NewMongoRepository create new repository
func NewMongoRepository(db *database.DB, conf *config.Config, logger *log.Logger) Repository {
	return &repo{
		database: db,
		config:   conf,
		log:      logger,
	}
}

func (r *repo) CreateIndexes() error {
	session := r.database.Session.Copy()
	defer session.Close()

	// set collection name
	collectionName := r.config.T.BeaconProxy.BeaconProxyTable

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
		{Key: []string{"-score"}},
		{Key: []string{"dst", "dst_network_uuid"}},
		{Key: []string{"src", "src_network_uuid"}},
		{Key: []string{"fqdn"}},
		{Key: []string{"-connection_count"}},
	}

	// create collection
	err := r.database.CreateCollection(collectionName, indexes)
	if err != nil {
		return err
	}

	return nil
}

//Upsert loops through every new fqdn requested from a proxy ....
func (r *repo) Upsert(proxyHostnameMap map[string]*Input, minTimestamp, maxTimestamp int64) {

	session := r.database.Session.Copy()
	defer session.Close()

	// Create the workers

	// stage 5 - write out results
	writerWorker := newWriter(
		r.config.T.BeaconProxy.BeaconProxyTable,
		r.database,
		r.config,
		r.log,
	)

	// stage 4 - perform the analysis
	analyzerWorker := newAnalyzer(
		minTimestamp,
		maxTimestamp,
		r.config.S.Rolling.CurrentChunk,
		r.database,
		r.config,
		writerWorker.collect,
		writerWorker.close,
	)

	// stage 3 - sort data
	sorterWorker := newSorter(
		r.database,
		r.config,
		analyzerWorker.collect,
		analyzerWorker.close,
	)

	// stage 2 - get and vet beacon details
	dissectorWorker := newDissector(
		int64(r.config.S.Strobe.ConnectionLimit),
		r.database,
		r.config,
		sorterWorker.collect,
		sorterWorker.close,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		dissectorWorker.start()
		sorterWorker.start()
		analyzerWorker.start()
		writerWorker.start()
	}

	// progress bar for troubleshooting
	p := mpb.New(mpb.WithWidth(20))
	bar := p.AddBar(int64(len(proxyHostnameMap)),
		mpb.PrependDecorators(
			decor.Name("\t[-] Proxy Beacon Analysis:", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over map entries (each hostname)
	for _, entry := range proxyHostnameMap {

		start := time.Now()

		// pass entry to dissector
		dissectorWorker.collect(entry)

		// progress bar increment
		bar.IncrBy(1, time.Since(start))

	}
	p.Wait()

	// start the closing cascade (this will also close the other channels)
	dissectorWorker.close()
}
