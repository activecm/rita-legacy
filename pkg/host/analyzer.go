package host

import (
	"github.com/activecm/rita-legacy/config"
	"github.com/activecm/rita-legacy/database"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"

	"sync"
)

type (
	//analyzer provides analysis of host records
	analyzer struct {
		chunk            int                        //current chunk (0 if not on rolling analysis)
		conf             *config.Config             // contains details needed to access MongoDB
		db               *database.DB               // provides access to MongoDB
		log              *log.Logger                // logger for writing out errors and warnings
		analyzedCallback func(database.BulkChanges) // called on each analyzed result
		closedCallback   func()                     // called when .close() is called and no more calls to analyzedCallback will be made
		analysisChannel  chan *Input                // holds unanalyzed data
		analysisWg       sync.WaitGroup             // wait for analysis to finish
	}
)

// newAnalyzer creates a new collector for gathering data
func newAnalyzer(chunk int, conf *config.Config, db *database.DB, log *log.Logger, analyzedCallback func(database.BulkChanges), closedCallback func()) *analyzer {
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

// collect sends a chunk of data to be analyzed
func (a *analyzer) collect(datum *Input) {
	a.analysisChannel <- datum
}

// close waits for the collector to finish
func (a *analyzer) close() {
	close(a.analysisChannel)
	a.analysisWg.Wait()
	a.closedCallback()
}

// start kicks off a new analysis thread
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

			totalUpdate := database.MergeBSONMaps(mainUpdate, blUpdate, connCountsUpdate)

			a.analyzedCallback(database.BulkChanges{
				a.conf.T.Structure.HostTable: []database.BulkChange{{
					Selector: datum.Host.BSONKey(),
					Update:   totalUpdate,
					Upsert:   true,
				}},
			})
		}
		a.analysisWg.Done()
	}()
}

// mainQuery sets the top level host information
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

// blQuery marks the given host as blacklisted or not
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

// connCountsQuery records the number of connections this host has been a part of
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
