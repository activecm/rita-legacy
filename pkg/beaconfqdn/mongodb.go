package beaconfqdn

import (
	"fmt"
	"runtime"
	"time"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/util"

	"github.com/briandowns/spinner"
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
		{Key: []string{"src", "fqdn", "src_network_uuid"}, Unique: true},
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

//Upsert first loops through every new host and determines which hostnames
//may have been contacted. Then it gathers the associated IPs for each of the
//hostnames, passing them onto the beacon analysis.
func (r *repo) Upsert(hostMap map[string]*host.Input, minTimestamp, maxTimestamp int64) {
	session := r.database.Session.Copy()
	defer session.Close()

	s := spinner.New(spinner.CharSets[36], 200*time.Millisecond)
	s.Prefix = "\t[-] Gathering FQDNs for Beacon Analysis ...\t"
	s.Start()

	// determine which hostnames need their fqdn beacon entries updated by
	// checking which hostnames are associated with the external IPs we saw in this import run.

	// NOTE: We used to feed the hostnamesMap into the dissector directly. In other words,
	// for each hostname, we would only aggregate the unique connections for the IPs which were included
	// in the DNS answers for current run of `rita import`. This had two downsides:
	// - if an internal host cached the only DNS response for an external host between RITA runs,
	//   the fqdn beacon would disappear. It should exist until the `hostnames` record rotates out via chunking.
	// - if we need to parse into the same chunk multiple times due to file size restrictions in the importer,
	//   we would never aggregate all of the unique connections across parse chunks if the internal hosts
	//   did not constantly issue DNS queries for the hosts they contacted. --LL
	affectedHostnames, err := r.affectedHostnameIPs(hostMap)
	if err != nil {
		r.log.WithError(err).Error("could not determine which hostnames need beacon data updates")
	}

	s.Stop()
	fmt.Println()

	if len(affectedHostnames) == 0 {
		fmt.Println("\t[!] No FQDN Beacon data to analyze")
		return
	}

	// Phase 1: Analysis

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
		r.log,
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

	// kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		dissectorWorker.start()
		sorterWorker.start()
		analyzerWorker.start()
		writerWorker.start()
	}

	// progress bar for troubleshooting
	p := mpb.New(mpb.WithWidth(20))
	bar := p.AddBar(int64(len(affectedHostnames)),
		mpb.PrependDecorators(
			decor.Name("\t[-] FQDN Beacon Analysis (1/2):", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over map entries (each hostname)
	for _, entry := range affectedHostnames {

		// check to make sure hostname has resolved ips, skip otherwise
		if len(entry.ResolvedIPs) > 0 {

			var dstList []bson.M
			for _, dst := range entry.ResolvedIPs {
				dstList = append(dstList, dst.AsDst().BSONKey())
			}

			input := &fqdnInput{
				FQDN:        entry.Host,
				DstBSONList: dstList,
				ResolvedIPs: entry.ResolvedIPs,
			}

			dissectorWorker.collect(input)
		}

		// progress bar increment
		bar.IncrBy(1)

	}

	p.Wait()

	// start the closing cascade (this will also close the other channels)
	dissectorWorker.close()

	// Phase 2: Summary

	// initialize a new writer for the summarizer
	writerWorker = newWriter(
		r.config.T.Structure.HostTable,
		r.database,
		r.config,
		r.log,
	)

	summarizerWorker := newSummarizer(
		r.config.S.Rolling.CurrentChunk,
		r.database,
		r.config,
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
	var localHosts []data.UniqueIP
	for _, entry := range hostMap {
		if entry.IsLocal {
			localHosts = append(localHosts, entry.Host)
		}
	}

	// add a progress bar for troubleshooting
	p = mpb.New(mpb.WithWidth(20))
	bar = p.AddBar(int64(len(localHosts)),
		mpb.PrependDecorators(
			decor.Name("\t[-] FQDN Beacon Analysis (2/2):", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over the local hosts that need to be summarized
	for _, localHost := range localHosts {
		summarizerWorker.collect(localHost)
		bar.IncrBy(1)
	}
	p.Wait()

	// start the closing cascade (this will also close the other channels)
	summarizerWorker.close()
}

// affectedHostnameIPs gathers all of the hostnames associated with the external IPs which generated
// traffic in this run. Each hostname entry is returned along with its list of associated resolved IPs.
func (r *repo) affectedHostnameIPs(hostMap map[string]*host.Input) ([]hostnameIPs, error) {

	// We can only submit around 200,000 hosts in a single query to MongoDB due to the 16MB document limit.
	// 1 Megabyte / Size of Unique IP BSON query ~= 200,000
	// 16 * 1024 * 1024 / (64 for IPv4, 83 for IPv6) = (262144 for IPv4, 202135 for IPv6)

	// provide a fast path for datasets with less than 200,000 hosts
	if len(hostMap) <= 200000 {
		return r.affectedHostnameIPsSimple(hostMap)
	}
	return r.affectedHostnameIPsChunked(hostMap)
}

// affectedHostnameIPsSimple implements affectedHostnameIPs but should only be called with hostMaps with
// roughly less than 200,000 external hosts.
func (r *repo) affectedHostnameIPsSimple(hostMap map[string]*host.Input) ([]hostnameIPs, error) {
	// preallocate externalHosts slice assuming at least half of the observed hosts are external
	// In most implementations of the Go runtime, when the array underlying a slice is
	// reallocated via append(), the runtime will double the size of the underlying array.
	// With this assumption in mind, we can assume that externalHosts will be reallocated
	// only one time at most.
	externalHosts := make([]data.UniqueIP, 0, len(hostMap)/2)
	var affectedHostnamesBuffer []hostnameIPs

	ssn := r.database.Session.Copy()
	defer ssn.Close()

	for _, host := range hostMap {
		if host.IsLocal {
			continue
		}
		externalHosts = append(externalHosts, host.Host)
	}
	reverseDNSagg := reverseDNSQueryWithIPs(externalHosts)
	externalHosts = nil // nolint ,we are freeing this potentially large array so the GC can claim it during MongoDB IO

	err := ssn.DB(r.database.GetSelectedDB()).C(r.config.T.DNS.HostnamesTable).
		Pipe(reverseDNSagg).AllowDiskUse().All(&affectedHostnamesBuffer)
	return affectedHostnamesBuffer, err
}

// affectedHostnameIPsChunked implements affectedHostnameIPs. It is slower than affectedHostnameIPs_simple
// and uses more RAM. However, unlike affectedHostnameIPs_simple, affectedHostnameIPs_chunked handles
// hostMaps of all sizes.
func (r *repo) affectedHostnameIPsChunked(hostMap map[string]*host.Input) ([]hostnameIPs, error) {
	// preallocate externalHosts slice assuming at least half of the observed hosts are external.
	// util.Min ensures that we don't preallocate more than the maximum allowed number of hosts in a 16MB BSON document.
	// In most implementations of the Go runtime, when the array underlying a slice is
	// reallocated via append(), the runtime will double the size of the underlying array.
	// With this assumption in mind, we can assume that externalHosts will be reallocated
	// only one time at most.
	externalHosts := make([]data.UniqueIP, 0, util.Min(200000, len(hostMap)/2))
	var affectedHostnamesBuffer []hostnameIPs

	ssn := r.database.Session.Copy()
	defer ssn.Close()

	// we will need to remove duplicate results from each query of 200,000 hosts, slowing down the process
	// and consuming more RAM

	// affectedHostnameIPMap maps hostnames to their respective ResolvedIPs
	// affectedHostnameIPMap is preallocated with the assumption that there are roughly
	// 10 IPs associated with each hostname in the set of logs.
	affectedHostnameIPMap := make(map[string][]data.UniqueIP, len(hostMap)/10)

	for _, host := range hostMap {
		if host.IsLocal {
			continue
		}

		externalHosts = append(externalHosts, host.Host)

		if len(externalHosts) == 200000 { // we've hit the limit, run the MongoDB query
			reverseDNSagg := reverseDNSQueryWithIPs(externalHosts)
			externalHosts = externalHosts[:0] // clear externalHosts to make room for the next chunk

			affectedHostnamesBuffer = nil
			err := ssn.DB(r.database.GetSelectedDB()).C(r.config.T.DNS.HostnamesTable).
				Pipe(reverseDNSagg).AllowDiskUse().All(&affectedHostnamesBuffer)
			if err != nil {
				return []hostnameIPs{}, err
			}

			for i := range affectedHostnamesBuffer {
				affectedHostnameIPMap[affectedHostnamesBuffer[i].Host] = affectedHostnamesBuffer[i].ResolvedIPs
			}
		}
	}

	if len(externalHosts) > 0 { // take care of running the MongoDB query for any leftover hosts
		reverseDNSagg := reverseDNSQueryWithIPs(externalHosts)
		externalHosts = nil // nolint , we are assigning nil here to allow the GC to free this potentially large array

		affectedHostnamesBuffer = nil
		err := ssn.DB(r.database.GetSelectedDB()).C(r.config.T.DNS.HostnamesTable).
			Pipe(reverseDNSagg).AllowDiskUse().All(&affectedHostnamesBuffer)
		if err != nil {
			return []hostnameIPs{}, err
		}

		for i := range affectedHostnamesBuffer {
			affectedHostnameIPMap[affectedHostnamesBuffer[i].Host] = affectedHostnamesBuffer[i].ResolvedIPs
		}
	}

	// convert the map into a slice
	affectedHostnamesBuffer = make([]hostnameIPs, 0, len(affectedHostnameIPMap))
	for host, ips := range affectedHostnameIPMap {
		affectedHostnamesBuffer = append(affectedHostnamesBuffer, hostnameIPs{
			Host:        host,
			ResolvedIPs: ips,
		})
	}
	return affectedHostnamesBuffer, nil
}

/*
db.getCollection('hostnames').aggregate([
	{"$match": { "$or": [
		{
			"dat.ips.ip": "104.16.107.25",
			"dat.ips.network_uuid": UUID("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		},
		{
			"dat.ips.ip" : "104.16.229.152",
			"dat.ips.network_uuid" : UUID("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		},
	]}},
	{"$project": {
		"host": 1,
		"dat.ips.ip": 1,
		"dat.ips.network_uuid": 1,
	}},
	{"$unwind": "$dat"},
	{"$unwind": "$dat.ips"},
	{"$group": {
		"_id": {
			"host": "$host",
			"ip": "$dat.ips.ip",
			"network_uuid": "$dat.ips.network_uuid",
		},
	}},
	{"$group": {
		"_id": "$_id.host",
		"ips": {"$push": {
			"ip": "$_id.ip",
			"network_uuid": "$_id.network_uuid",
		}},
	}}
])

reverseDNSQueryWithIPs returns a MongoDB aggregation which returns the hostnames associated with the given
UniqueIPs. Additionally, all of the IPs associated with each hostname are returned.
*/
func reverseDNSQueryWithIPs(uniqueIPs []data.UniqueIP) []bson.M {
	uniqueIPBsonSelectors := make([]bson.M, 0, len(uniqueIPs))
	for i := range uniqueIPs {
		uniqueIPBsonSelectors = append(uniqueIPBsonSelectors, bson.M{
			"dat.ips.ip":           uniqueIPs[i].IP,
			"dat.ips.network_uuid": uniqueIPs[i].NetworkUUID,
		})
	}
	return []bson.M{
		{"$match": bson.M{"$or": uniqueIPBsonSelectors}},
		{"$project": bson.M{
			"host":                 1,
			"dat.ips.ip":           1,
			"dat.ips.network_uuid": 1,
		}},
		{"$unwind": "$dat"},
		{"$unwind": "$dat.ips"},
		{"$group": bson.M{
			"_id": bson.M{
				"host":         "$host",
				"ip":           "$dat.ips.ip",
				"network_uuid": "$dat.ips.network_uuid",
			},
		}},
		{"$group": bson.M{
			"_id": "$_id.host",
			"ips": bson.M{"$push": bson.M{
				"ip":           "$_id.ip",
				"network_uuid": "$_id.network_uuid",
			}},
		}},
	}
}
