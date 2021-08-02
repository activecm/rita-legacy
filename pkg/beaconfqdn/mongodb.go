package beaconfqdn

import (
	"runtime"
	"time"

	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/briandowns/spinner"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

type repo struct {
	res *resources.Resources
	min int64
	max int64
}

//NewMongoRepository create new repository
func NewMongoRepository(res *resources.Resources) Repository {
	min, max, _ := res.MetaDB.GetTSRange(res.DB.GetSelectedDB())
	return &repo{
		res: res,
		min: min,
		max: max,
	}
}

func (r *repo) CreateIndexes() error {
	session := r.res.DB.Session.Copy()
	defer session.Close()

	// set collection name
	collectionName := r.res.Config.T.BeaconFQDN.BeaconFQDNTable

	// check if collection already exists
	names, _ := session.DB(r.res.DB.GetSelectedDB()).CollectionNames()

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
	err := r.res.DB.CreateCollection(collectionName, indexes)
	if err != nil {
		return err
	}

	return nil
}

//Upsert first loops through every new host and determines which hostnames
//may have been contacted. Then it gathers the associated IPs for each of the
//hostnames, passing them onto the beacon analysis.
func (r *repo) Upsert(hostMap map[string]*host.Input) {
	session := r.res.DB.Session.Copy()
	defer session.Close()

	// Create the workers

	// stage 5 - write out results
	writerWorker := newWriter(
		r.res.Config.T.BeaconFQDN.BeaconFQDNTable,
		r.res.DB,
		r.res.Config,
		r.res.Log,
	)

	// stage 4 - perform the analysis
	analyzerWorker := newAnalyzer(
		r.min,
		r.max,
		r.res.Config.S.Rolling.CurrentChunk,
		r.res.DB,
		r.res.Config,
		writerWorker.collect,
		writerWorker.close,
	)

	// stage 3 - sort data
	sorterWorker := newSorter(
		r.res.DB,
		r.res.Config,
		analyzerWorker.collect,
		analyzerWorker.close,
	)

	// stage 2 - get and vet beacon details
	dissectorWorker := newDissector(
		int64(r.res.Config.S.Strobe.ConnectionLimit),
		r.res.DB,
		r.res.Config,
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
		r.res.Log.WithError(err).Error("could not determine which hostnames need beacon data updates")
	}

	s.Stop()

	// progress bar for troubleshooting
	p := mpb.New(mpb.WithWidth(20))
	bar := p.AddBar(int64(len(affectedHostnames)),
		mpb.PrependDecorators(
			decor.Name("\t[-] FQDN Beacon Analysis:", decor.WC{W: 30, C: decor.DidentRight}),
			decor.CountersNoUnit(" %d / %d ", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	// loop over map entries (each hostname)
	for _, entry := range affectedHostnames {

		start := time.Now()

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

		// progress bar increment
		bar.IncrBy(1, time.Since(start))

	}

	p.Wait()

	// start the closing cascade (this will also close the other channels)
	dissectorWorker.close()
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
	} else {
		return r.affectedHostnameIPsChunked(hostMap)
	}
}

// affectedHostnameIPsSimple implements affectedHostnameIPs but should only be called with hostMaps with
// roughly less than 200,000 external hosts.
func (r *repo) affectedHostnameIPsSimple(hostMap map[string]*host.Input) ([]hostnameIPs, error) {
	var externalHosts []data.UniqueIP = make([]data.UniqueIP, 0, len(hostMap)/2)
	var affectedHostnamesBuffer []hostnameIPs

	ssn := r.res.DB.Session.Copy()
	defer ssn.Close()

	for _, host := range hostMap {
		if host.IsLocal {
			continue
		}
		externalHosts = append(externalHosts, host.Host)
	}
	reverseDNSagg := reverseDNSQueryWithIPs(externalHosts)
	externalHosts = nil // nolint ,we are freeing this potentially large array so the GC can claim it during MongoDB IO

	err := ssn.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.DNS.HostnamesTable).
		Pipe(reverseDNSagg).All(&affectedHostnamesBuffer)
	return affectedHostnamesBuffer, err
}

// affectedHostnameIPsChunked implements affectedHostnameIPs. It is slower than affectedHostnameIPs_simple
// and uses more RAM. However, unlike affectedHostnameIPs_simple, affectedHostnameIPs_chunked handles
// hostMaps of all sizes.
func (r *repo) affectedHostnameIPsChunked(hostMap map[string]*host.Input) ([]hostnameIPs, error) {
	var externalHosts []data.UniqueIP = make([]data.UniqueIP, 0, len(hostMap)/2)
	var affectedHostnamesBuffer []hostnameIPs

	ssn := r.res.DB.Session.Copy()
	defer ssn.Close()

	// we will need to remove duplicate results from each query of 200,000 hosts, slowing down the process
	// and consuming more RAM
	var affectedHostnameIPMap map[string][]data.UniqueIP = make(map[string][]data.UniqueIP, len(hostMap)/10)

	for _, host := range hostMap {
		if host.IsLocal {
			continue
		}

		if len(externalHosts) == 200000 {
			reverseDNSagg := reverseDNSQueryWithIPs(externalHosts)
			externalHosts = nil

			err := ssn.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.DNS.HostnamesTable).
				Pipe(reverseDNSagg).All(&affectedHostnamesBuffer)
			if err != nil {
				return []hostnameIPs{}, err
			}

			for i := range affectedHostnamesBuffer {
				if _, ok := affectedHostnameIPMap[affectedHostnamesBuffer[i].Host]; !ok {
					affectedHostnameIPMap[affectedHostnamesBuffer[i].Host] = affectedHostnamesBuffer[i].ResolvedIPs
				}
			}
			affectedHostnamesBuffer = nil
		}

		externalHosts = append(externalHosts, host.Host)
	}

	if len(externalHosts) > 0 {
		reverseDNSagg := reverseDNSQueryWithIPs(externalHosts)
		externalHosts = nil // nolint , we are assigning nil here to allow the GC to free this potentially large array

		err := ssn.DB(r.res.DB.GetSelectedDB()).C(r.res.Config.T.DNS.HostnamesTable).
			Pipe(reverseDNSagg).All(&affectedHostnamesBuffer)
		if err != nil {
			return []hostnameIPs{}, err
		}

		for i := range affectedHostnamesBuffer {
			if _, ok := affectedHostnameIPMap[affectedHostnamesBuffer[i].Host]; !ok {
				affectedHostnameIPMap[affectedHostnamesBuffer[i].Host] = affectedHostnamesBuffer[i].ResolvedIPs
			}
		}
		affectedHostnamesBuffer = nil
	}

	affectedHostnamesBuffer = make([]hostnameIPs, 0, len(affectedHostnamesBuffer))
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
	var uniqueIPBsonSelectors []bson.M = make([]bson.M, 0, len(uniqueIPs))
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
