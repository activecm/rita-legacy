package host

import (
	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/data"
	"github.com/globalsign/mgo/bson"
	"strconv"
	"sync"
)

type (
	//analyzer : structure for host analysis
	analyzer struct {
		chunk            int            //current chunk (0 if not on rolling analysis)
		chunkStr         string         //current chunk (0 if not on rolling analysis)
		db               *database.DB   // provides access to MongoDB
		conf             *config.Config // contains details needed to access MongoDB
		analyzedCallback func(update)   // called on each analyzed result
		closedCallback   func()         // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Input    // holds unanalyzed data
		analysisWg       sync.WaitGroup // wait for analysis to finish
	}
)

//newAnalyzer creates a new collector for gathering data
func newAnalyzer(chunk int, db *database.DB, conf *config.Config, analyzedCallback func(update), closedCallback func()) *analyzer {
	return &analyzer{
		chunk:            chunk,
		chunkStr:         strconv.Itoa(chunk),
		db:               db,
		conf:             conf,
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
			// blacklisted flag
			blacklisted := false

			// check if blacklisted destination
			blCount, _ := ssn.DB(a.conf.S.Blacklisted.BlacklistDatabase).C("ip").Find(bson.M{"index": datum.Host.IP}).Count()
			if blCount > 0 {
				blacklisted = true
			}

			// find maximum dns query count in domain[count] map
			// this represents the total count of dns queries made by the domain who
			// was most queried by this host
			var maxDNSQCount int64
			maxDNSQCount = 0
			if len(datum.MaxDNSQueryCount) > 0 {
				for _, count := range datum.MaxDNSQueryCount {
					if count > maxDNSQCount {
						maxDNSQCount = count
					}
				}
			}

			// update src of connection in hosts table
			if datum.IP4 {
				var output update
				newRecordFlag := false
				type hostRes struct {
					CID int `bson:"cid"`
				}

				var res2 []hostRes

				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(datum.Host.BSONKey()).All(&res2)

				if !(len(res2) > 0) {
					newRecordFlag = true
					// fmt.Println("host no results", res2, datum.Host)
				} else {

					if res2[0].CID != a.chunk {
						// fmt.Println("host existing", a.chunk, res2, datum.Host)
						newRecordFlag = true
					}
				}

				output = standardQuery(a.chunk, a.chunkStr, datum.Host, datum.IsLocal, datum.IP4, datum.IP4Bin, datum.MaxDuration, maxDNSQCount, datum.UntrustedAppConnCount, datum.CountSrc, datum.CountDst, blacklisted, newRecordFlag)

				// set to writer channel
				a.analyzedCallback(output)

			}

		}
		a.analysisWg.Done()
	}()
}

//standardQuery ...
func standardQuery(chunk int, chunkStr string, ip data.UniqueIP, local bool, ip4 bool, ip4bin int64, maxdur float64, maxDNSQCount int64, untrustedACC int64, countSrc int, countDst int, blacklisted bool, newFlag bool) update {
	var output update

	// create query
	query := bson.M{
		"$set": bson.M{
			"blacklisted":  blacklisted,
			"cid":          chunk,
			"local":        local,
			"ipv4":         ip4,
			"ipv4_binary":  ip4bin,
			"network_name": ip.NetworkName,
		},
	}
	if newFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"count_src":       countSrc,
				"count_dst":       countDst,
				"max_dns_query_count": maxDNSQCount,
				"upps_count":      untrustedACC,
				"cid":             chunk,
			}}

		// create selector for output ,
		output.query = query
		output.selector = ip.BSONKey()

	} else {

		query["$inc"] = bson.M{
			"dat.$.count_src":       countSrc,
			"dat.$.count_dst":       countDst,
			"dat.$.max_dns_query_count": maxDNSQCount,
			"dat.$.upps_count":      untrustedACC,
		}

		// create selector for output
		output.query = query
		output.selector = ip.BSONKey()
		output.selector["dat.cid"] = chunk
	}

	return output
}
