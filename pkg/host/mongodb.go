package host

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

	coll := session.DB(r.database.GetSelectedDB()).C(r.config.T.Structure.HostTable)

	// create hosts collection
	// Desired indexes
	indexes := []mgo.Index{
		{Key: []string{"ip", "network_uuid"}, Unique: true},
		{Key: []string{"local"}},
		{Key: []string{"ipv4_binary"}},
		{Key: []string{"dat.mdip.ip", "dat.mdip.network_uuid"}},
		{Key: []string{"dat.mbdst.ip", "dat.mbdst.network_uuid"}},
		{Key: []string{"dat.max_dns.query"}},
		{Key: []string{"dat.mbfqdn"}},
		{Key: []string{"dat.mbproxy"}},
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
func (r *repo) Upsert(hostMap map[string]*Input) {

	// 1st Phase: Analysis

	// Create the workers
	writerWorker := newWriter(r.config.T.Structure.HostTable, r.database, r.config, r.log)

	analyzerWorker := newAnalyzer(
		r.config.S.Rolling.CurrentChunk,
		r.config,
		r.database,
		r.log,
		writerWorker.collect,
		writerWorker.close,
	)

	// kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		analyzerWorker.start()
		writerWorker.start()
	}

	// progress bar for troubleshooting
	p := mpb.New(mpb.WithWidth(20))
	bar := p.AddBar(int64(len(hostMap)),
		mpb.PrependDecorators(
			decor.Name("\t[-] Host Analysis (1/2):", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over map entries
	for _, entry := range hostMap {
		analyzerWorker.collect(entry)
		bar.IncrBy(1)
	}
	p.Wait()

	// start the closing cascade (this will also close the other channels)
	analyzerWorker.close()

	// 2nd Phase: Summarize

	// initialize a new writer for the summarizer
	writerWorker = newWriter(r.config.T.Structure.HostTable, r.database, r.config, r.log)
	summarizerWorker := newSummarizer(
		r.config.S.Rolling.CurrentChunk,
		r.config,
		r.database,
		r.log,
		writerWorker.collect,
		writerWorker.close,
	)

	// kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		summarizerWorker.start()
		writerWorker.start()
	}

	// get local hosts only for the summary
	var localHosts []*Input
	for _, entry := range hostMap {
		if entry.IsLocal {
			localHosts = append(localHosts, entry)
		}
	}

	// progress bar for troubleshooting
	p = mpb.New(mpb.WithWidth(20))
	bar = p.AddBar(int64(len(localHosts)),
		mpb.PrependDecorators(
			decor.Name("\t[-] Host Analysis (2/2):", decor.WC{W: 30, C: decor.DidentRight}),
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
