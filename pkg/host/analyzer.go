package host

import (
	"strconv"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
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
		analysisChannel  chan *IP       // holds unanalyzed data
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
		analysisChannel:  make(chan *IP),
	}
}

//collect sends a chunk of data to be analyzed
func (a *analyzer) collect(data *IP) {
	a.analysisChannel <- data
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

		for data := range a.analysisChannel {
			// blacklisted flag
			blacklisted := false

			//TODO[AGENT]: Use new name for IP.Host (Input.IP) for checking ip blacklist

			// check if blacklisted destination
			blCount, _ := ssn.DB(a.conf.S.Blacklisted.BlacklistDatabase).C("ip").Find(bson.M{"index": data.Host}).Count()
			if blCount > 0 {
				blacklisted = true
			}

			// update src of connection in hosts table
			if data.IP4 {
				var output update
				newRecordFlag := false
				type hostRes struct {
					CID int `bson:"cid"`
				}

				var res2 []hostRes

				//TODO[AGENT]: Use new name for IP.Host (Input.IP) and Input.NetworkID for checking if ip in host table already

				_ = ssn.DB(a.db.GetSelectedDB()).C(a.conf.T.Structure.HostTable).Find(bson.M{"ip": data.Host}).All(&res2)

				if !(len(res2) > 0) {
					newRecordFlag = true
					// fmt.Println("host no results", res2, data.Host)
				} else {

					if res2[0].CID != a.chunk {
						// fmt.Println("host existing", a.chunk, res2, data.Host)
						newRecordFlag = true
					}
				}

				output = standardQuery(a.chunk, a.chunkStr, data.Host, data.IsLocal, data.IP4, data.IP4Bin, data.MaxDuration, data.TXTQueryCount, data.UntrustedAppConnCount, data.CountSrc, data.CountDst, blacklisted, newRecordFlag)

				// set to writer channel
				a.analyzedCallback(output)

			}

		}
		a.analysisWg.Done()
	}()
}

//standardQuery ...
func standardQuery(chunk int, chunkStr string, ip string, local bool, ip4 bool, ip4bin int64, maxdur float64, txtQCount int64, untrustedACC int64, countSrc int, countDst int, blacklisted bool, newFlag bool) update {
	var output update

	//TODO[AGENT]: Integrate UniqueIP NetworkID/ Network Name into host collection aggregation "query"

	// create query
	query := bson.M{
		"$set": bson.M{
			"blacklisted": blacklisted,
			"cid":         chunk,
			"local":       local,
			"ipv4":        ip4,
			"ipv4_binary": ip4bin,
		},
	}

	if newFlag {

		query["$push"] = bson.M{
			"dat": bson.M{
				"count_src":       countSrc,
				"count_dst":       countDst,
				"txt_query_count": txtQCount,
				"upps_count":      untrustedACC,
				"cid":             chunk,
			}}

		// create selector for output ,
		output.query = query
		output.selector = bson.M{"ip": ip}

	} else {

		query["$inc"] = bson.M{
			"dat.$.count_src":       countSrc,
			"dat.$.count_dst":       countDst,
			"dat.$.txt_query_count": txtQCount,
			"dat.$.upps_count":      untrustedACC,
		}

		// create selector for output
		output.query = query
		output.selector = bson.M{"ip": ip, "dat.cid": chunk}
	}

	return output
}
