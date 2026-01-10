package sniconn

import (
	"runtime"

	"github.com/activecm/rita-legacy/config"
	"github.com/activecm/rita-legacy/database"
	"github.com/activecm/rita-legacy/pkg/data"
	"github.com/activecm/rita-legacy/pkg/host"
	"github.com/activecm/rita-legacy/util"
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

// NewMongoRepository bundles the given resources for updating MongoDB with SNI connection data
func NewMongoRepository(db *database.DB, conf *config.Config, logger *log.Logger) Repository {
	return &repo{
		database: db,
		config:   conf,
		log:      logger,
	}
}

// CreateIndexes creates indexes for the SNIconn collection
func (r *repo) CreateIndexes() error {
	session := r.database.Session.Copy()
	defer session.Close()

	// set collection name
	collectionName := r.config.T.Structure.SNIConnTable

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
		{Key: []string{"src", "fqdn", "src_network_uuid"}, Unique: true},
		{Key: []string{"src", "src_network_uuid"}},
		{Key: []string{"fqdn"}},
		{Key: []string{"dat.http.count"}},
		{Key: []string{"dat.tls.count"}},
	}

	// create collection
	err := r.database.CreateCollection(collectionName, indexes)
	if err != nil {
		return err
	}

	return nil
}

// Upsert records the given sni connection data in MongoDB. Summaries are
// created for the given local hosts in MongoDB.
func (r *repo) Upsert(tlsMap map[string]*TLSInput, httpMap map[string]*HTTPInput, zeekUIDMap map[string]*data.ZeekUIDRecord, hostMap map[string]*host.Input) {

	// Phase 1: Analysis

	// Merge separate input maps from the parser
	linkedInputMap := linkInputMaps(tlsMap, httpMap, zeekUIDMap)

	// Create the workers for analysis
	writerWorker := database.NewBulkWriter(r.database, r.config, r.log, true, "sniconn")

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
	bar := p.AddBar(int64(len(linkedInputMap)),
		mpb.PrependDecorators(
			decor.Name("\t[-] SNI Connection Analysis:", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over map entries
	for _, entry := range linkedInputMap {
		analyzerWorker.collect(entry)
		bar.IncrBy(1)
	}
	p.Wait()

	// start the closing cascade (this will also close the other channels)
	analyzerWorker.close()
}

func linkInputMaps(tlsMap map[string]*TLSInput, httpMap map[string]*HTTPInput, zeekUIDMap map[string]*data.ZeekUIDRecord) map[string]*linkedInput {
	linkedMap := make(map[string]*linkedInput, len(tlsMap)+len(httpMap))
	for tlsKey, tlsValue := range tlsMap {
		if _, ok := linkedMap[tlsKey]; !ok {
			linkedMap[tlsKey] = &linkedInput{}
		}

		var tlsZeekRecords []*data.ZeekUIDRecord
		for _, zeekUID := range tlsValue.ZeekUIDs {
			if zeekRecord, ok := zeekUIDMap[zeekUID]; ok {
				tlsZeekRecords = append(tlsZeekRecords, zeekRecord)
			}
		}

		linkedMap[tlsKey].TLS = tlsValue
		linkedMap[tlsKey].TLSZeekRecords = tlsZeekRecords
	}

	for httpKey, httpValue := range httpMap {
		if _, ok := linkedMap[httpKey]; !ok {
			linkedMap[httpKey] = &linkedInput{}
		}

		var httpZeekRecords []*data.ZeekUIDRecord
		for _, zeekUID := range httpValue.ZeekUIDs {
			if zeekRecord, ok := zeekUIDMap[zeekUID]; ok {
				httpZeekRecords = append(httpZeekRecords, zeekRecord)
			}
		}

		linkedMap[httpKey].HTTP = httpValue
		linkedMap[httpKey].HTTPZeekRecords = httpZeekRecords
	}

	return linkedMap
}
