package remover

import (
	"fmt"
	"runtime"

	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo/bson"
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

	// first we need to use the entries being removed from hostnames to reduce the
	// subdomain count in exploded dns. This is done so we don't have to keep Unique
	// long lists of subdomains, and is the only special-case deletion
	err := r.reduceDNSSubCount(cid)
	if err != nil {
		return fmt.Errorf("\t[!] Failed to remove update exploded dns collection for removal: %v", err)
	}
	err = r.removeOutdatedCIDs(cid)
	if err != nil {
		return fmt.Errorf("\t[!] Failed to remove outdated documents from database")
	}

	return nil
}

func (r *remover) reduceDNSSubCount(cid int) error {
	ssn := r.res.DB.Session.Copy()
	defer ssn.Close()

	//Create the workers
	writerWorker := newUpdater(
		cid,
		r.res.DB,
		r.res.Config,
		r.res.Log,
	)

	analyzerWorker := newAnalyzer(
		cid,
		r.res.DB,
		r.res.Config,
		writerWorker.collectUpdater,
		writerWorker.closeUpdater,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		analyzerWorker.start()
		writerWorker.startUpdater()
	}

	// check if this query string has already been parsed to add to the subdomain count by checking
	// if the whole string is already in the hostname table.
	var res struct {
		Host string `bson:"host"`
	}

	hostnamesIter := ssn.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.DNS.HostnamesTable).Find(bson.M{"cid": cid}).Iter()

	for hostnamesIter.Next(&res) {
		analyzerWorker.collect(res.Host)
	}

	// start the closing cascade (this will also close the other channels)
	analyzerWorker.close()

	return nil
}

// removeOutdatedCIDs will remove every outdated document, as well as outdated chunks
// within current documents
func (r *remover) removeOutdatedCIDs(cid int) error {

	// list of all analysis modules (including hostnames and exploded dns, because
	// they still need to have this done, the previous loop was for updating existing
	// documents due to to the special case in how that data is updated and stored.
	modules := []string{
		r.res.Config.T.Beacon.BeaconTable,
		r.res.Config.T.Structure.HostTable,
		r.res.Config.T.Structure.UniqueConnTable,
		r.res.Config.T.DNS.ExplodedDNSTable,
		r.res.Config.T.DNS.HostnamesTable,
		r.res.Config.T.UserAgent.UserAgentTable,
	}

	//Create the workers
	writerWorker := newCIDRemover(
		cid,
		r.res.DB,
		r.res.Config,
		r.res.Log,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		writerWorker.startCIDRemover()
	}

	// loop over map entries
	for _, entry := range modules {
		writerWorker.collectCIDRemover(entry)
	}

	// close writer channel
	writerWorker.closeCIDRemover()

	return nil
}
