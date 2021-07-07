package beacon

import (
	"runtime"
	"time"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/uconn"
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
	collectionName := r.config.T.Beacon.BeaconTable

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
		{Key: []string{"src", "dst", "src_network_uuid", "dst_network_uuid"}, Unique: true},
		{Key: []string{"src", "src_network_uuid"}},
		{Key: []string{"dst", "dst_network_uuid"}},
		{Key: []string{"-connection_count"}},
	}

	// create collection
	err := r.database.CreateCollection(collectionName, indexes)
	if err != nil {
		return err
	}

	return nil
}

//Upsert loops through every new uconn ....
func (r *repo) Upsert(uconnMap map[string]*uconn.Input, minTimestamp, maxTimestamp int64) {

	//Create the workers
	writerWorker := newWriter(
		r.config.T.Beacon.BeaconTable,
		r.database,
		r.config,
		r.log,
	)

	analyzerWorker := newAnalyzer(
		minTimestamp,
		maxTimestamp,
		r.config.S.Rolling.CurrentChunk,
		r.database,
		r.config,
		writerWorker.collect,
		writerWorker.close,
	)

	sorterWorker := newSorter(
		r.database,
		r.config,
		analyzerWorker.collect,
		analyzerWorker.close,
	)

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
	bar := p.AddBar(int64(len(uconnMap)),
		mpb.PrependDecorators(
			decor.Name("\t[-] Beacon Analysis:", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over map entries
	for _, entry := range uconnMap {
		start := time.Now()
		dissectorWorker.collect(entry)
		bar.IncrBy(1, time.Since(start))
	}
	p.Wait()

	// start the closing cascade (this will also close the other channels)
	dissectorWorker.close()
}
