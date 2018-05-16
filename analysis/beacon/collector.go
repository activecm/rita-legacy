package beacon

import (
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/datatypes/data"
	"github.com/activecm/rita/datatypes/structure"
	"gopkg.in/mgo.v2/bson"
)

type (
	collector struct {
		db                  *database.DB
		conf                *config.Config
		connectionThreshold int                        // the minimum number of connections to be considered a beacon
		collectedCallback   func(*beaconAnalysisInput) // called on each collected set of connections
		collectChannel      chan string                // holds ip addresses
		collectWg           sync.WaitGroup             // wait for collection to finish
	}
)

func newCollector(db *database.DB, conf *config.Config, connectionThreshold int,
	collectedCallback func(*beaconAnalysisInput)) *collector {
	return &collector{
		db:                  db,
		conf:                conf,
		connectionThreshold: connectionThreshold,
		collectedCallback:   collectedCallback,
		collectChannel:      make(chan string),
	}
}

func (c *collector) collect(srcHost string) {
	c.collectChannel <- srcHost
}

func (c *collector) flush() {
	close(c.collectChannel)
	c.collectWg.Wait()
}

func (c *collector) start() {
	c.collectWg.Add(1)
	go func() {
		session := c.db.Session.Copy()
		defer session.Close()
		host, more := <-c.collectChannel
		for more {
			//grab all destinations related with this host
			var uconn structure.UniqueConnection
			destIter := session.DB(c.db.GetSelectedDB()).
				C(c.conf.T.Structure.UniqueConnTable).
				Find(bson.M{"src": host}).Iter()

			for destIter.Next(&uconn) {
				//skip the connection pair if they are under the threshold
				if uconn.ConnectionCount < c.connectionThreshold {
					continue
				}

				//create our new input
				newInput := &beaconAnalysisInput{
					uconnID: uconn.ID,
					src:     uconn.Src,
					dst:     uconn.Dst,
				}

				//Grab connection data
				var conn data.Conn
				connIter := session.DB(c.db.GetSelectedDB()).
					C(c.conf.T.Structure.ConnTable).
					Find(bson.M{"id_orig_h": uconn.Src, "id_resp_h": uconn.Dst}).
					Iter()

				for connIter.Next(&conn) {
					newInput.ts = append(newInput.ts, conn.Ts)
					newInput.origIPBytes = append(newInput.origIPBytes, conn.OriginIPBytes)
				}
				c.collectedCallback(newInput)
			}
			host, more = <-c.collectChannel
		}
		c.collectWg.Done()
	}()
}
