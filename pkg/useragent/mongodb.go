package useragent

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

func (r *repo) Upsert(userAgentMap map[string]*Input) {
	//Create the workers
	writerWorker := newWriter(
		r.database,
		r.config,
		r.log,
	)

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
		start := time.Now()
		//Mongo Index key is limited to a size of 1024 https://docs.mongodb.com/v3.4/reference/limits/#index-limitations
		//  so if the key is too large, we should cut it back, this is rough but
		//  works. Figured 800 allows some wiggle room, while also not being too large
		if len(entry.Name) > 1024 {
			entry.Name = entry.Name[:800]
		}
		analyzerWorker.collect(entry)
		bar.IncrBy(1, time.Since(start))
	}

	p.Wait()

	// start the closing cascade (this will also close the other channels)
	analyzerWorker.close()
}
