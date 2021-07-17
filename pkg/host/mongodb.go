package host

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

	coll := session.DB(r.database.GetSelectedDB()).C(r.config.T.Structure.HostTable)

	// create hosts collection
	// Desired indexes
	indexes := []mgo.Index{
		{Key: []string{"ip", "network_uuid"}, Unique: true},
		{Key: []string{"local"}},
		{Key: []string{"ipv4_binary"}},
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

	//Create the workers
	writerWorker := newWriter(r.config.T.Structure.HostTable, r.database, r.config, r.log)

	analyzerWorker := newAnalyzer(
		r.config.S.Rolling.CurrentChunk,
		r.config,
		r.database,
		r.log,
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
	bar := p.AddBar(int64(len(hostMap)),
		mpb.PrependDecorators(
			decor.Name("\t[-] Host Analysis:", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over map entries
	for _, entry := range hostMap {
		start := time.Now()
		analyzerWorker.collect(entry)
		bar.IncrBy(1, time.Since(start))
	}
	p.Wait()

	// start the closing cascade (this will also close the other channels)
	analyzerWorker.close()
}
