package beacon

import (
	"fmt"
	"runtime"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/host"
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

// NewMongoRepository create new repository
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

// Upsert derives beacon statistics from the given unique connections and creates summaries
// for the given local hosts. The results are pushed to MongoDB.
func (r *repo) Upsert(uconnMap map[string]*uconn.Input, hostMap map[string]*host.Input, minTimestamp, maxTimestamp int64) {

	//Create the workers
	writerWorker := database.NewBulkWriter(
		r.database,
		r.config,
		r.log,
		true,
		"beacon",
	)

	analyzerWorker := newAnalyzer(
		minTimestamp,
		maxTimestamp,
		r.config.S.Rolling.CurrentChunk,
		r.database,
		r.config,
		r.log,
		writerWorker.Collect,
		writerWorker.Close,
	)

	sorterWorker := newSorter(
		r.database,
		r.config,
		analyzerWorker.collect,
		analyzerWorker.close,
	)

	siphonWorker := newSiphon(
		int64(r.config.S.Strobe.ConnectionLimit),
		r.config.S.Rolling.CurrentChunk,
		r.database,
		r.config,
		r.log,
		writerWorker.Collect,
		sorterWorker.collect,
		sorterWorker.close,
	)

	dissectorWorker := newDissector(
		int64(r.config.S.Strobe.ConnectionLimit),
		r.config.S.Rolling.CurrentChunk,
		r.database,
		r.config,
		siphonWorker.collect,
		siphonWorker.close,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		dissectorWorker.start()
		siphonWorker.start()
		sorterWorker.start()
		analyzerWorker.start()
		writerWorker.Start()
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
		dissectorWorker.collect(entry)
		bar.IncrBy(1)
	}
	p.Wait()

	// start the closing cascade (this will also close the other channels)
	dissectorWorker.close()

	// Phase 2: Summary

	// grab the local hosts we have seen during the current analysis period
	// get local hosts only for the summary
	var localHosts []data.UniqueIP
	for _, entry := range hostMap {
		if entry.IsLocal {
			localHosts = append(localHosts, entry.Host)
		}
	}

	// skip the summarize phase if there are no local hosts to summarize
	if len(localHosts) == 0 {
		fmt.Println("\t[!] Skipping Beacon Aggregation: No Internal Hosts")
		return
	}

	// initialize a new writer for the summarizer
	writerWorker = database.NewBulkWriter(r.database, r.config, r.log, true, "beacon")
	summarizerWorker := newSummarizer(
		r.config.S.Rolling.CurrentChunk,
		r.database,
		r.config,
		r.log,
		writerWorker.Collect,
		writerWorker.Close,
	)

	// kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		summarizerWorker.start()
		writerWorker.Start()
	}

	// add a progress bar for troubleshooting
	p = mpb.New(mpb.WithWidth(20))
	bar = p.AddBar(int64(len(localHosts)),
		mpb.PrependDecorators(
			decor.Name("\t[-] Beacon Aggregation:", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over the local hosts that need to be summarized
	for _, localHost := range localHosts {
		summarizerWorker.collect(localHost)
		bar.IncrBy(1)
	}

	p.Wait()

	// start the closing cascade (this will also close the other channels)
	summarizerWorker.close()
}
