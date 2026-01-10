package useragent

import (
	"fmt"
	"runtime"

	"github.com/activecm/rita-legacy/config"
	"github.com/activecm/rita-legacy/database"
	"github.com/activecm/rita-legacy/pkg/data"
	"github.com/activecm/rita-legacy/pkg/host"
	"github.com/activecm/rita-legacy/util"

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

// NewMongoRepository bundles the given resources for updating MongoDB with rare signature data
func NewMongoRepository(db *database.DB, conf *config.Config, logger *log.Logger) Repository {
	return &repo{
		database: db,
		config:   conf,
		log:      logger,
	}
}

// CreateIndexes creates indexes for the useragent collection
func (r *repo) CreateIndexes() error {
	session := r.database.Session.Copy()
	defer session.Close()

	// set collection name
	collectionName := r.config.T.UserAgent.UserAgentTable

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
		{Key: []string{"user_agent"}, Unique: true},
		{Key: []string{"dat.seen"}},
		{Key: []string{"dat.orig_ips.ip", "dat.orig_ips.network_uuid"}},
	}

	// create collection
	err := r.database.CreateCollection(collectionName, indexes)
	if err != nil {
		return err
	}

	return nil
}

// Upsert records the given useragent data in MongoDB
func (r *repo) Upsert(userAgentMap map[string]*Input, hostMap map[string]*host.Input) {

	// 1st Phase: Analysis

	for _, entry := range userAgentMap {
		//Mongo Index key is limited to a size of 1024 https://docs.mongodb.com/v3.4/reference/limits/#index-limitations
		//  so if the key is too large, we should cut it back, this is rough but
		//  works. Figured 800 allows some wiggle room, while also not being too large
		if len(entry.Name) > 1024 {
			entry.Name = entry.Name[:800]
		}
	}

	// Create the workers
	writerWorker := database.NewBulkWriter(
		r.database,
		r.config,
		r.log,
		true,
		"useragent",
	)

	analyzerWorker := newAnalyzer(
		r.config.S.Rolling.CurrentChunk,
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
	bar := p.AddBar(int64(len(userAgentMap)),
		mpb.PrependDecorators(
			decor.Name("\t[-] UserAgent Analysis:", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over map entries
	for _, entry := range userAgentMap {
		analyzerWorker.collect(entry)
		bar.IncrBy(1)
	}

	p.Wait()

	// start the closing cascade (this will also close the other channels)
	analyzerWorker.close()

	// 2nd Phase: Summarize

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
		fmt.Println("\t[!] Skipping UserAgent Aggregation: No Internal Hosts")
		return
	}

	// initialize a new writer for the summarizer
	writerWorker = database.NewBulkWriter(r.database, r.config, r.log, true, "useragent")
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

	// progress bar for troubleshooting
	p = mpb.New(mpb.WithWidth(20))
	bar = p.AddBar(int64(len(localHosts)),
		mpb.PrependDecorators(
			decor.Name("\t[-] UserAgent Aggregation:", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over map entries
	for _, entry := range localHosts {
		summarizerWorker.collect(entry)
		bar.IncrBy(1)
	}
	p.Wait()

	// start the closing cascade (this will also close the other channels)
	summarizerWorker.close()
}
