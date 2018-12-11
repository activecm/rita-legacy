package blacklist

import (
	"fmt"
	"runtime"

	bl "github.com/activecm/rita-bl"

	"github.com/activecm/rita/datatypes/blacklist"
	"github.com/activecm/rita/datatypes/structure"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
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
func buildBlacklistedIPs(ips *mgo.Iter, res *resources.Resources,
	blHandle *bl.Blacklist, bufferSize int, source bool) {
	//create session to write to
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	fmt.Println("-- buildBlacklisted IPs --")

	// create the actual collection
	collectionName := res.Config.T.Blacklisted.DestIPsTable
	collectionKeys := []mgo.Index{
		{Key: []string{"$hashed:ip"}},
	}
	err := res.DB.CreateCollection(collectionName, collectionKeys)
	if err != nil {
		res.Log.Error("Failed: ", collectionName, err.Error())
		return
	}

	//choose the output collection
	// var outputCollection *mgo.Collection
	// if source {
	// 	outputCollection = ssn.DB(res.DB.GetSelectedDB()).C(
	// 		res.Config.T.Blacklisted.SourceIPsTable,
	// 	)
	// } else {
	// 	outputCollection = ssn.DB(res.DB.GetSelectedDB()).C(
	// 		res.Config.T.Blacklisted.DestIPsTable,
	// 	)
	// }

	checkRitaBlacklistIPs(ips, blHandle, bufferSize, res)

	// //create type for communicating rita-bl results
	// resultsChannel := make(resultsChan)
	//
	// //kick off the checking process
	// go checkRitaBlacklistIPs(ips, blHandle, bufferSize, resultsChannel, res)
	//
	// //results are maps from ip addresses to arrays of their respective results
	// for results := range resultsChannel {
	// 	//loop over the map
	// 	for ipAddr, individualResults := range results {
	// 		//if the ip address has blacklist results
	// 		if len(individualResults) > 0 {
	// 			blIP := blacklist.BlacklistedIP{IP: ipAddr}
	// 			for _, result := range individualResults {
	// 				blIP.Lists = append(blIP.Lists, result.List)
	// 			}
	// 			err := fillBlacklistedIP(
	// 				&blIP,
	// 				res.DB.GetSelectedDB(),
	// 				res.Config.T.Structure.UniqueConnTable,
	// 				res.Config.T.Structure.HostTable,
	// 				ssn,
	// 				source,
	// 			)
	// 			if err != nil {
	// 				res.Log.WithFields(log.Fields{
	// 					"err": err.Error(),
	// 					"ip":  ipAddr,
	// 					"db":  res.DB.GetSelectedDB(),
	// 				}).Error("could not aggregate info on blacklisted IP")
	// 				continue
	// 			}
	// 			outputCollection.Insert(&blIP)
	//
	// 		}
	// 	}
	// }
}

func checkRitaBlacklistIPs(ips *mgo.Iter, blHandle *bl.Blacklist,
	bufferSize int, res *resources.Resources) {
	fmt.Println("-- checkRitaBlacklist IPs --")
	i := 0
	count := 0

	//Create the workers
	writerWorker := newWriter(res.DB, res.Config)
	analyzerWorker := newAnalyzer(
		res.DB,
		writerWorker.write, writerWorker.close,
	)

	//kick off the threaded goroutines
	for i := 0; i < util.Max(1, runtime.NumCPU()/2); i++ {
		analyzerWorker.start()
		writerWorker.start()
	}

	var uconnRes struct {
		IP string `bson:"ip"`
	}
	//read in bufferSize entries and check them. Then ship them off to the writer
	// var buff = make([]ipAggregateResult, bufferSize)
	for ips.Next(&uconnRes) {
		// fmt.Println(buff[i])
		// if i == bufferSize-1 {
		// 	//excuse the memory hacking to get better performance
		// 	//We need the buffer to be of type ipAggregateResult for
		// 	//proper marshalling, but we need strings for rita-blacklist.
		// 	//The underlying memory for ipAggregateResult is that of a string
		// 	//since it is the only field in the struct.
		// 	//So we can safely view buff as an array of strings using a
		// 	//reinterpret cast. Then, we can dereference the pointer to the array
		// 	//and use the variadic syntax to pass the array to CheckEntries.
		// 	indexesArray := (*[]string)(unsafe.Pointer(&buff))
		// 	resultsChannel <- blHandle.CheckEntries(
		// 		list.BlacklistedIPType, (*indexesArray)...,
		// 	)
		// 	//reset the buffer
		// 	i = 0
		// }

		newInput := &blacklist.AnalysisInput{
			IP: uconnRes.IP,
		}
		analyzerWorker.analyze(newInput)
		i++
		count++
	}
	//if there are left overs in the buffer
	// if i != 0 {
	// 	buffSlice := buff[:i]
	// 	indexesArray := (*[]string)(unsafe.Pointer(&buffSlice))
	// 	resultsChannel <- blHandle.CheckEntries(
	// 		list.BlacklistedIPType, (*indexesArray)...,
	// 	)
	// }
	// close(resultsChannel)

	fmt.Println("total: ", count)
}

// fillBlacklistedIP tallies the total number of bytes and connections
// made to each blacklisted IP. It stores this information in the blIP
// parameter. The source parameter is true if the blacklisted IP initiated
// the connections or false if the blacklisted IP received the connections.
func fillBlacklistedIP(blIP *blacklist.BlacklistedIP, db, uconnCollection string,
	hostCollection string, ssn *mgo.Session, source bool) error {
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

	// Loop through uconn to add up the total number of bytes and connections
	// Also update the non-blacklist side of the connection's host collection entry
	for uniqueConnections.Next(&uconn) {
		totalBytes += uconn.TotalBytes
		totalConnections += uconn.ConnectionCount
		uniqueConnCount++

		// For every set of connections made to a blacklisted IP, we want to
		// keep track of how much data (# of conns and # of bytes) was sent
		// or received by the internal IP.
		if source {
			// If the blacklisted IP initiated the connection, then bl_in_count
			// holds the number of unique blacklisted IPs connected to the given
			// host.
			// bl_sum_avg_bytes adds the average number of bytes over all
			// individual connections between these two systems. This is an
			// indication of how much data was transferred overall but not take
			// into account the number of connections.
			// bl_total_bytes adds the total number of bytes sent over all
			// individual connections between the two systems.
			ssn.DB(db).C(hostCollection).Update(
				bson.M{"ip": uconn.Dst},
				bson.D{
					{"$inc", bson.M{"bl_in_count": 1}},
					{"$inc", bson.M{"bl_sum_avg_bytes": uconn.AverageBytes}},
					{"$inc", bson.M{"bl_total_bytes": uconn.TotalBytes}},
				})
		} else {
			// If the internal system initiated the connection, then bl_out_count
			// holds the number of unique blacklisted IPs the given host contacted.
			// bl_sum_avg_bytes and bl_total_bytes are the same as above.
			ssn.DB(db).C(hostCollection).Update(
				bson.M{"ip": uconn.Src},
				bson.D{
					{"$inc", bson.M{"bl_out_count": 1}},
					{"$inc", bson.M{"bl_sum_avg_bytes": uconn.AverageBytes}},
					{"$inc", bson.M{"bl_total_bytes": uconn.TotalBytes}},
				})
		}
	}

	blIP.Connections = totalConnections
	blIP.UniqueConnections = uniqueConnCount
	blIP.TotalBytes = totalBytes

	return nil
}
