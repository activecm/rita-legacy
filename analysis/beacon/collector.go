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
	// collector collects Conn records into groups based on destination given
	// a source host
	collector struct {
		db                  *database.DB               // provides access to MongoDB
		conf                *config.Config             // contains details needed to access MongoDB
		connectionThreshold int                        // the minimum number of connections to be considered a beacon
		collectedCallback   func(*beaconAnalysisInput) // called on each collected set of connections
		collectChannel      chan string                // holds ip addresses
		collectWg           sync.WaitGroup             // wait for collection to finish
	}
)

// newCollector creates a new collector for creating beaconAnalysisInput objects
// which group the given source, a detected destination, and all of their
// connection analysis details (timestamps, data sizes, etc.)
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

// collect queues a host for collection
// Note: this function may block
func (c *collector) collect(srcHost string) {
	c.collectChannel <- srcHost
}

// flush waits for the collection threads to finish
func (c *collector) flush() {
	close(c.collectChannel)
	c.collectWg.Wait()
}

// start kicks off a new collection thread
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
					//TODO: Test. Currently none of the test cases mark proto, so they
					//pass

					//filter out unestablished connections
					//We expect at least SYN ACK SYN-ACK [FIN ACK FIN ACK/ RST]
					if conn.Proto == "tcp" && conn.OriginPackets+conn.ResponsePackets <= 3 {
						continue
					}

					newInput.ts = append(newInput.ts, conn.Ts)
					newInput.origIPBytes = append(newInput.origIPBytes, conn.OriginIPBytes)
				}

				//filtering may have reduced the amount of connections
				//check again if we should skip this unique connection
				if len(newInput.ts) < c.connectionThreshold {
					continue
				}

				c.collectedCallback(newInput)
			}
			host, more = <-c.collectChannel
		}
		c.collectWg.Done()
	}()
}
