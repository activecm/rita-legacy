package remover

import (
	"fmt"
	"runtime"

	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
)

type remover struct {
	res *resources.Resources
}

//NewMongoRemover create new repository
func NewMongoRemover(res *resources.Resources) Repository {
	return &remover{
		res: res,
	}
}

//Upsert loops through every new uconn ....
func (r *remover) Remove(cid int) error {

	fmt.Println("\t[-] Removing matching chunk: ", cid+1)

	modules := []string{
		r.res.Config.T.Beacon.BeaconTable,
		r.res.Config.T.Structure.HostTable,
		r.res.Config.T.Structure.UniqueConnTable,
		r.res.Config.T.DNS.ExplodedDNSTable,
		r.res.Config.T.DNS.HostnamesTable,
		r.res.Config.T.UserAgent.UserAgentTable,
	}

	//Create the workers
	writerWorker := newWriter(
		cid,
		r.res.DB,
		r.res.Config,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		writerWorker.start()
	}

	// loop over map entries
	for _, entry := range modules {
		// 	start := time.Now()
		writerWorker.collect(entry)
		// 	bar.IncrBy(1, time.Since(start))
	}
	// p.Wait()
	return nil
}
