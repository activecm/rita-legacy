package remover

import (
	"github.com/activecm/rita/resources"
)

type remover struct {
	res *resources.Resources
}

//NewMongoRepository create new repository
func NewMongoRemover(res *resources.Resources) Repository {
	return &remover{
		res: res,
	}
}

//Upsert loops through every new uconn ....
func (r *repo) Remove() {
	// //Create the workers
	// writerWorker := newWriter(
	// 	r.res.Config.T.Beacon.BeaconTable,
	// 	r.res.DB,
	// 	r.res.Config,
	// )
	//
	// analyzerWorker := newAnalyzer(
	// 	r.res.DB,
	// 	r.res.Config,
	// 	writerWorker.collect,
	// 	writerWorker.close,
	// )
	//
	// sorterWorker := newSorter(
	// 	r.res.DB,
	// 	r.res.Config,
	// 	analyzerWorker.collect,
	// 	analyzerWorker.close,
	// )
	//
	// dissectorWorker := newDissector(
	// 	int64(r.res.Config.S.Strobe.ConnectionLimit),
	// 	r.res.DB,
	// 	r.res.Config,
	// 	sorterWorker.collect,
	// 	sorterWorker.close,
	// )
	//
	// //kick off the threaded goroutines
	// for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
	// 	dissectorWorker.start()
	// 	sorterWorker.start()
	// 	analyzerWorker.start()
	// 	writerWorker.start()
	// }
	//
	// // progress bar for troubleshooting
	// p := mpb.New(mpb.WithWidth(20))
	// bar := p.AddBar(int64(len(uconnMap)),
	// 	mpb.PrependDecorators(
	// 		decor.Name("\t[-] Beacon Analysis:", decor.WC{W: 30, C: decor.DidentRight}),
	// 		decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
	// 	),
	// 	mpb.AppendDecorators(decor.Percentage()),
	// )
	//
	// // loop over map entries
	// for _, entry := range uconnMap {
	// 	start := time.Now()
	// 	dissectorWorker.collect(entry)
	// 	bar.IncrBy(1, time.Since(start))
	// }
	// p.Wait()
}
