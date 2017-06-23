package blacklist

import (
	"unsafe"

	bl "github.com/ocmdev/rita-blacklist2"
	"github.com/ocmdev/rita-blacklist2/list"
	"github.com/ocmdev/rita/database"
	data "github.com/ocmdev/rita/datatypes/blacklist"
	"github.com/ocmdev/rita/datatypes/structure"
	log "github.com/sirupsen/logrus"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type ipAggregateResult struct {
	IP string `bson:"ip"`
}

func getUniqueIPFromUconnPipeline(field string) []bson.D {
	//nolint: vet
	return []bson.D{
		{
			{"$project", bson.D{
				{"ip", "$" + field},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", bson.D{
					{"ip", "$ip"},
				}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"ip", "$_id.ip"},
			}},
		},
	}
}

//buildBlacklistedIPs builds a set of blacklisted ips from the
//iterator provided, the system config, a handle to rita-blacklist,
//a buffer of ips to check at a time, and a boolean designating
//whether or not the ips are connection sources or destinations
func buildBlacklistedIPs(ips *mgo.Iter, res *database.Resources,
	blHandle *bl.Blacklist, bufferSize int, source bool) {
	//create session to write to
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	//choose the output collection
	var outputCollection *mgo.Collection
	if source {
		outputCollection = ssn.DB(res.DB.GetSelectedDB()).C(
			res.System.BlacklistedConfig.SourceIPsTable,
		)
	} else {
		outputCollection = ssn.DB(res.DB.GetSelectedDB()).C(
			res.System.BlacklistedConfig.DestIPsTable,
		)
	}

	//create type for communicating rita-bl results
	resultsChannel := make(resultsChan)

	//kick off the checking process
	go checkRitaBlacklistIPs(ips, blHandle, bufferSize, resultsChannel)

	//results are maps from ip addresses to arrays of their respective results
	for results := range resultsChannel {
		//loop over the map
		for ipAddr, individualResults := range results {
			//if the ip address has blacklist results
			if len(individualResults) > 0 {
				blIP := data.BlacklistedIP{IP: ipAddr}
				err := fillBlacklistedIP(
					&blIP,
					res.DB.GetSelectedDB(),
					res.System.StructureConfig.UniqueConnTable,
					ssn,
					source,
				)
				if err != nil {
					res.Log.WithFields(log.Fields{
						"err": err.Error(),
						"ip":  ipAddr,
						"db":  res.DB.GetSelectedDB(),
					}).Error("could not aggregate info on blacklisted IP")
					continue
				}
				outputCollection.Insert(&blIP)
			}
		}
	}
}

func checkRitaBlacklistIPs(ips *mgo.Iter, blHandle *bl.Blacklist,
	bufferSize int, resultsChannel resultsChan) {
	i := 0
	//read in bufferSize entries and check them. Then ship them off to the writer
	var buff = make([]ipAggregateResult, bufferSize)
	for ips.Next(&buff[i]) {
		if i == bufferSize-1 {
			//excuse the memory hacking to get better performance
			//We need the buffer to be of type ipAggregateResult for
			//proper marshalling, but we need strings for rita-blacklist.
			//The underlying memory for ipAggregateResult is that of a string
			//since it is the only field in the struct.
			//So we can safely view buff as an array of strings using a
			//reinterpret cast. Then, we can dereference the pointer to the array
			//and use the variadic syntax to pass the array to CheckEntries.
			indexesArray := (*[]string)(unsafe.Pointer(&buff))
			resultsChannel <- blHandle.CheckEntries(
				list.BlacklistedIPType, (*indexesArray)...,
			)
			//reset the buffer
			i = 0
		}
		i++
	}
	//if there are left overs in the buffer
	if i != 0 {
		buffSlice := buff[:i]
		indexesArray := (*[]string)(unsafe.Pointer(&buffSlice))
		resultsChannel <- blHandle.CheckEntries(
			list.BlacklistedIPType, (*indexesArray)...,
		)
	}
	close(resultsChannel)
}

func fillBlacklistedIP(blIP *data.BlacklistedIP, db, uconnCollection string,
	ssn *mgo.Session, source bool) error {
	var connQuery bson.M
	if source {
		connQuery = bson.M{"src": blIP.IP}
	} else {
		connQuery = bson.M{"dst": blIP.IP}
	}

	var totalBytes int
	var totalConnections int
	var uniqueConnCount int
	uniqueConnections := ssn.DB(db).C(uconnCollection).Find(connQuery).Iter()
	var uconn structure.UniqueConnection
	for uniqueConnections.Next(&uconn) {
		totalBytes += uconn.TotalBytes
		totalConnections += uconn.ConnectionCount
		uniqueConnCount++
	}
	blIP.Connections = totalConnections
	blIP.UniqueConnections = uniqueConnCount
	blIP.TotalBytes = totalBytes

	return nil
}
