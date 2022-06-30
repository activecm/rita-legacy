package host

import (
	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"

	"strings"
	"sync"
)

type (
	//analyzer provides analysis of host records
	analyzer struct {
		chunk            int            //current chunk (0 if not on rolling analysis)
		conf             *config.Config // contains details needed to access MongoDB
		db               *database.DB   // provides access to MongoDB
		log              *log.Logger    // logger for writing out errors and warnings
		analyzedCallback func(update)   // called on each analyzed result
		closedCallback   func()         // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Input    // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for gathering data
func newAnalyzer(chunk int, conf *config.Config, db *database.DB, log *log.Logger, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		conf:             conf,
		log:              log,
		db:               db,
		analyzedCallback: analyzedCallback,
		closedCallback:   closedCallback,
		analysisChannel:  make(chan *Input),
	}
}

//collect sends a chunk of data to be analyzed
func (a *analyzer) collect(datum *Input) {
	a.analysisChannel <- datum
}

//close waits for the collector to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

//start kicks off a new analysis thread
func (a *analyzer) start() {
	a.analysisWg.Add(1)
	go func() {
		ssn := a.db.Session.Copy()
		defer ssn.Close()

		for datum := range a.analysisChannel {
			if !datum.IP4 { // we currently only handle IPv4 addresses
				continue
			}

			mainUpdate := mainQuery(datum, a.chunk)

			blUpdate, err := blQuery(datum, ssn, a.conf.S.Blacklisted.BlacklistDatabase) // TODO: Move to BL package
			if err != nil {
				a.log.WithFields(log.Fields{
					"Module": "host",
					"Data":   datum.Host,
				}).Error(err)
			}

			connCountsUpdate := connCountsQuery(datum, a.chunk)
			expDNSUpdate := explodedDNSQuery(datum, a.chunk)

			totalUpdate := database.MergeBSONMaps(mainUpdate, blUpdate, connCountsUpdate, expDNSUpdate)

			a.analyzedCallback(update{
				selector: datum.Host.BSONKey(),
				query:    totalUpdate,
			})
		}
		a.analysisWg.Done()
	}()
}

//mainQuery sets the top level host information
func mainQuery(datum *Input, chunk int) bson.M {
	return bson.M{
		"$set": bson.M{
			"cid":          chunk,
			"local":        datum.IsLocal,
			"ipv4":         datum.IP4,
			"ipv4_binary":  datum.IP4Bin,
			"network_name": datum.Host.NetworkName,
		},
	}
}

//blQuery marks the given host as blacklisted or not
func blQuery(datum *Input, ssn *mgo.Session, blDB string) (bson.M, error) {
	// check if blacklisted destination
	blCount, err := ssn.DB(blDB).C("ip").Find(bson.M{"index": datum.Host.IP}).Count()
	blacklisted := blCount > 0

	return bson.M{
		"$set": bson.M{
			"blacklisted": blacklisted,
		},
	}, err
}

//connCountsQuery records the number of connections this host has been a part of
func connCountsQuery(datum *Input, chunk int) bson.M {
	return bson.M{
		"$push": bson.M{
			"dat": bson.M{
				"$each": []bson.M{{
					"count_src":  datum.CountSrc,
					"count_dst":  datum.CountDst,
					"upps_count": datum.UntrustedAppConnCount,
					"cid":        chunk,
				}},
			},
		},
	}
}

//explodedDNSQuery records the result of the individual host's exploded dns analysis for this chunk
func explodedDNSQuery(datum *Input, chunk int) bson.M {
	if len(datum.DNSQueryCount) == 0 {
		return bson.M{}
	}

	// update the host record with the new exploded dns results
	explodedDNSEntries := buildExplodedDNSArray(datum.DNSQueryCount)
	return bson.M{
		"$push": bson.M{
			"dat": bson.M{
				"$each": []bson.M{{
					"exploded_dns": explodedDNSEntries,
					"cid":          chunk,
				}},
			},
		},
	}
}

//buildExplodedDNSArray generates exploded dns query results given how many times each full fqdn
//was queried. Returns the results as an array for MongoDB compatibility
func buildExplodedDNSArray(dnsQueryCounts map[string]int64) []explodedDNS {
	// make a new map to store the exploded dns query->count data
	explodedDNSMap := make(map[string]int64)

	for domain := range dnsQueryCounts {
		// split name on periods
		split := strings.Split(domain, ".")

		// we will not count the very last item, because it will be either all or
		// a part of the tlds. This means that something like ".co.uk" will still
		// not be fully excluded, but it will greatly reduce the complexity for the
		// most common tlds
		max := len(split) - 1

		for i := 1; i <= max; i++ {
			// parse domain which will be the part we are on until the end of the string
			entry := strings.Join(split[max-i:], ".")
			explodedDNSMap[entry]++
		}
	}

	// put exploded dns map into mongo format so that we can push the entire
	// exploded dns map data into the database in one go
	var explodedDNSEntries []explodedDNS
	for domain, count := range explodedDNSMap {
		var explodedDNSEntry explodedDNS
		explodedDNSEntry.Query = domain
		explodedDNSEntry.Count = count
		explodedDNSEntries = append(explodedDNSEntries, explodedDNSEntry)
	}
	return explodedDNSEntries
}
