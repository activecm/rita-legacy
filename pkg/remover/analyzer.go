package remover

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
		analysisChannel  chan string    // holds unanalyzed data
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
		analysisChannel:  make(chan string),
	}
}

//collect sends a group of domains to be analyzed
func (a *analyzer) collect(data string) {
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

		for data := range a.analysisChannel {

			a.reduceDNSSubCount(data)

		}

		a.analysisWg.Done()
	}()
}

func (a *analyzer) reduceDNSSubCount(name string) {

	// split name on periods
	split := strings.Split(name, ".")

	// we will not count the very last item, because it will be either all or
	// a part of the tlds. This means that something like ".co.uk" will still
	// not be fully excluded, but it will greatly reduce the complexity for the
	// most common tlds
	max := len(split) - 1

	for i := 1; i <= max; i++ {
		// parse domain which will be the part we are on until the end of the string
		entry := strings.Join(split[max-i:], ".")

		// in some of these strings, the empty space will get counted as a domain,
		// this was an issue in the old version of exploded dns and caused inaccuracies
		if (entry == "") || (entry == "in-addr.arpa") {
			break
		}

		// set up writer output
		var output update

		output.query = bson.M{
			"$inc": bson.M{
				"subdomain_count": -1,
			},
		}

		// create selector for output
		output.selector = bson.M{"domain": entry}

		output.collection = a.conf.T.DNS.ExplodedDNSTable

		// set to writer channel
		a.analyzedCallback(output)

	}

}
