package beacon

import (
	"github.com/activecm/rita/datatypes/data"
	"github.com/activecm/rita/datatypes/structure"
	"gopkg.in/mgo.v2/bson"
)

// collect grabs all src, dst pairs and their connection data
func (t *Beacon) collect() {
	session := t.res.DB.Session.Copy()
	defer session.Close()
	host, more := <-t.collectChannel
	for more {
		//grab all destinations related with this host
		var uconn structure.UniqueConnection
		destIter := session.DB(t.db).
			C(t.res.Config.T.Structure.UniqueConnTable).
			Find(bson.M{"src": host}).Iter()

		for destIter.Next(&uconn) {
			//skip the connection pair if they are under the threshold
			if uconn.ConnectionCount < t.defaultConnThresh {
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
			connIter := session.DB(t.db).
				C(t.res.Config.T.Structure.ConnTable).
				Find(bson.M{"id_orig_h": uconn.Src, "id_resp_h": uconn.Dst}).
				Iter()

			for connIter.Next(&conn) {
				newInput.ts = append(newInput.ts, conn.Ts)
				newInput.origIPBytes = append(newInput.origIPBytes, conn.OriginIPBytes)
			}
			t.analysisChannel <- newInput
		}
		host, more = <-t.collectChannel
	}
	t.collectWg.Done()
}
