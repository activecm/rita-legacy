package remover

import (
	"fmt"
	"runtime"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/util"
	"github.com/globalsign/mgo/bson"

	log "github.com/sirupsen/logrus"
)

type remover struct {
	database *database.DB
	config   *config.Config
	log      *log.Logger
}

//NewMongoRemover create new remover. Handles removing outdated data by chunk ID.
func NewMongoRemover(db *database.DB, conf *config.Config, logger *log.Logger) Repository {
	return &remover{
		database: db,
		config:   conf,
		log:      logger,
	}
}

//Upsert loops through every new uconn ....
func (r *remover) Remove(cid int) error {

	fmt.Println("\t[-] Removing matching chunk: ", cid)

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
	ssn := r.database.Session.Copy()
	defer ssn.Close()

	//Create the workers
	writerWorker := newUpdater(
		cid,
		r.database,
		r.config,
		r.log,
	)

	analyzerWorker := newAnalyzer(
		cid,
		r.database,
		r.config,
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

	hostnamesIter := ssn.DB(r.database.GetSelectedDB()).C(r.config.T.DNS.HostnamesTable).Find(bson.M{"cid": cid}).Iter()

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
		r.config.T.Beacon.BeaconTable,
		r.config.T.BeaconFQDN.BeaconFQDNTable,
		r.config.T.Structure.HostTable,
		r.config.T.Structure.UniqueConnTable,
		r.config.T.DNS.ExplodedDNSTable,
		r.config.T.DNS.HostnamesTable,
		r.config.T.Cert.CertificateTable,
		r.config.T.UserAgent.UserAgentTable,
	}

	//Create the workers
	writerWorker := newCIDRemover(
		cid,
		r.database,
		r.config,
		r.log,
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
