package hostname

import (
	"strconv"
	"strings"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
)

type (
	//analyzer : structure for exploded dns analysis
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

//newAnalyzer creates a new collector for parsing hostnames
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

//collect sends a group of domains to be analyzed
func (a *analyzer) collect(data *Input) {
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

			// in some of these strings, the empty space will get counted as a domain,
			// this was an issue in the old version of exploded dns and caused inaccuracies
			if (data.Host == "") || (strings.HasSuffix(data.Host, "in-addr.arpa")) {
				continue
			}

			// set blacklisted Flag
			blacklistFlag := false

			// check ip against blacklist
			blCount, _ := ssn.DB(a.conf.S.Blacklisted.BlacklistDatabase).C("hostname").Find(bson.M{"index": data.Host}).Count()
			// check if hostname is blacklisted
			if blCount > 0 {
				blacklistFlag = true
			}

			// set up writer output
			var output update

			// create query
			if blacklistFlag {
				// flag as blacklisted if blacklisted
				output.query = bson.M{
					"$push": bson.M{
						"dat": bson.M{
							"ips":     data.ResolvedIPs,
							"src_ips": data.ClientIPs,
							"cid":     a.chunk,
						},
					},
					"$set": bson.M{
						"blacklisted": true,
						"cid":         a.chunk,
					},
				}
			} else {
				output.query = bson.M{
					"$push": bson.M{
						"dat": bson.M{
							"ips":     data.ResolvedIPs,
							"src_ips": data.ClientIPs,
							"cid":     a.chunk,
						},
					},
					"$set": bson.M{"cid": a.chunk},
				}
			}

			// create selector for output
			output.selector = bson.M{"host": data.Host}

			// set to writer channel
			a.analyzedCallback(output)

		}

		a.analysisWg.Done()
	}()
}
