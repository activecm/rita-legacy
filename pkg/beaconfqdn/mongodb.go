package beaconfqdn

import (
	"runtime"
	"time"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/util"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
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
	collectionName := r.config.T.BeaconFQDN.BeaconFQDNTable

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

//Upsert loops through every new hostname ....
func (r *repo) Upsert(hostnameMap map[string]*hostname.Input, minTimestamp, maxTimestamp int64) {
	session := r.database.Session.Copy()
	defer session.Close()

	// Create the workers

	// stage 5 - write out results
	writerWorker := newWriter(
		r.config.T.BeaconFQDN.BeaconFQDNTable,
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
	bar := p.AddBar(int64(len(hostnameMap)),
		mpb.PrependDecorators(
			decor.Name("\t[-] FQDN Beacon Analysis:", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over map entries (each hostname)
	for _, entry := range hostnameMap {

		start := time.Now()

		// check to make sure hostname has resolved ips, skip otherwise
		if len(entry.ResolvedIPs) > 0 {

			var dstList []bson.M
			for _, dst := range entry.ResolvedIPs {
				dstList = append(dstList, dst.AsDst().BSONKey())
			}

			input := &hostname.FqdnInput{
				FQDN:        entry.Host,
				DstBSONList: dstList,
				ResolvedIPs: entry.ResolvedIPs,
			}

			dissectorWorker.collect(input)
		}

		// progress bar increment
		bar.IncrBy(1, time.Since(start))

	}
	p.Wait()

	// start the closing cascade (this will also close the other channels)
	dissectorWorker.close()
}
