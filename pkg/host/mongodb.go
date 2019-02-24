package host

import (
	"runtime"
	"time"

	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

type repo struct {
	res *resources.Resources
}

//NewMongoRepository create new repository
func NewMongoRepository(res *resources.Resources) Repository {
	return &repo{
		res: res,
	}
}

func (r *repo) CreateIndexes() error {
	session := r.res.DB.Session.Copy()
	defer session.Close()

	coll := session.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.Structure.HostTable)

	// create hosts collection
	// Desired indexes
	indexes := []mgo.Index{
		{Key: []string{"ip"}, Unique: true},
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
func (r *repo) Upsert(hostMap map[string]*IP) {

	//Create the workers
	writerWorker := newWriter(r.res.Config.T.Structure.HostTable, r.res.DB, r.res.Config)

	analyzerWorker := newAnalyzer(
		r.res.Config.S.Bro.CurrentChunk,
		r.res.DB,
		r.res.Config,
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
		// if entry.Host == "10.55.100.106" {
		//
		// }
		start := time.Now()
		analyzerWorker.collect(entry)
		bar.IncrBy(1, time.Since(start))
	}
	p.Wait()
}
