package beaconfqdn

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/hostname"
	"github.com/globalsign/mgo/bson"
)

type (
	accumulator struct {
		db                  *database.DB              // provides access to MongoDB
		conf                *config.Config            // contains details needed to access MongoDB
		accumulatedCallback func(*hostname.FqdnInput) // called on each analyzed result
		closedCallback      func()                    // called when .close() is called and no more calls to analyzedCallback will be made
		accumulateChannel   chan *hostname.Input      // holds unanalyzed data
		accumulateWg        sync.WaitGroup            // wait for analysis to finish
	}
)

//newAccumulator creates a new collector for gathering data
func newAccumulator(db *database.DB, conf *config.Config, accumulatedCallback func(*hostname.FqdnInput), closedCallback func()) *accumulator {
	return &accumulator{
		db:                  db,
		conf:                conf,
		accumulatedCallback: accumulatedCallback,
		closedCallback:      closedCallback,
		accumulateChannel:   make(chan *hostname.Input),
	}
}

//collect sends a chunk of data to be analyzed
func (c *accumulator) collect(entry *hostname.Input) {
	c.accumulateChannel <- entry
}

//close waits for the collector to finish
func (c *accumulator) close() {
	close(c.accumulateChannel)
	c.accumulateWg.Wait()
	c.closedCallback()
}

//start kicks off a new analysis thread
func (c *accumulator) start() {
	c.accumulateWg.Add(1)
	go func() {
		ssn := c.db.Session.Copy()
		defer ssn.Close()

		for entry := range c.accumulateChannel {
			// create resolved dst array for match query
			var dstList []bson.M
			for _, dst := range entry.ResolvedIPs {
				dstList = append(dstList, dst.DstBSONKey())
			}

			// create match query
			srcMatchQuery := []bson.M{
				{"$match": bson.M{
					"$or": dstList,
				}},
				{"$project": bson.M{
					"src":              1,
					"src_network_uuid": 1,
					"src_network_name": 1,
				}},
			}

			// get all src ips that connected to the resolved ips
			var srcRes []data.UniqueSrcIP

			// execute query
			_ = ssn.DB(c.db.GetSelectedDB()).C(c.conf.T.Structure.UniqueConnTable).Pipe(srcMatchQuery).AllowDiskUse().All(&srcRes)

			// for each src that connected to a resolved ip...
			for _, src := range srcRes {

				input := &hostname.FqdnInput{
					Src:         src,
					FQDN:        entry.Host,
					DstBSONList: dstList,
					ResolvedIPs: entry.ResolvedIPs,
				}

				c.accumulatedCallback(input)
			}

		}
		c.accumulateWg.Done()
	}()
}
