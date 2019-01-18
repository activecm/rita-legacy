package structure

import (
	"fmt"
	"log"
	"net"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/datatypes/structure"
	"github.com/activecm/rita/resources"
	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// BuildHostsCollection builds the 'host' collection from the `uconn` collection.
func BuildHostsCollection(res *resources.Resources) {

	// verify if hosts collection was already created at import time
	names, err1 := res.DB.Session.DB(res.DB.GetSelectedDB()).CollectionNames()
	if err1 != nil {
		res.Log.Error("Failed to get coll names: ", err1)
		return
	}

	for _, name := range names {
		if name == res.Config.T.Structure.HostTable {
			log.Printf("\t\t[>] Host collection already exists!")
			return
		}
	}

	// Name of source collection which will be aggregated into the new collection
	sourceCollectionName := res.Config.T.Structure.UniqueConnTable

	// Name of the new collection
	newCollectionName := res.Config.T.Structure.HostTable

	// Desired indexes
	keys := []mgo.Index{
		{Key: []string{"ip"}, Unique: true},
		{Key: []string{"local"}},
		{Key: []string{"ipv4"}},
		{Key: []string{"ipv4_binary"}},
	}

	// Aggregate it!
	errorCheck := res.DB.CreateCollection(newCollectionName, keys)
	if errorCheck != nil {
		res.Log.Error("Failed: ", newCollectionName, errorCheck)
		return
	}

	getHosts(res, res.Config, sourceCollectionName, newCollectionName)
}

//getHosts aggregates the individual hosts from the source collection and
//labels them as private or public as well as ipv4 or ipv6. The aggregation
//includes padding for a binary encoding of the ip address.
func getHosts(res *resources.Resources, conf *config.Config, sourceCollection string, targetCollection string) {

	session := res.DB.Session.Copy()
	defer session.Close()

	// Aggregation to populate the hosts collection
	// nolint: vet
	uconnsFindQuery := []bson.D{
		{
			{"$project", bson.D{
				{"hosts", []interface{}{
					bson.D{
						{"ip", "$src"},
						{"local", "$local_src"},
						{"src", true},
						{"max_duration", "$max_duration"},
					},
					bson.D{
						{"ip", "$dst"},
						{"local", "$local_dst"},
						{"dst", true},
						{"max_duration", "$max_duration"},
					},
				}},
			}},
		},
		{
			{"$unwind", "$hosts"},
		},
		{
			{"$group", bson.D{
				{"_id", "$hosts.ip"},
				{"local", bson.D{
					{"$first", "$hosts.local"},
				}},
				{"src", bson.D{
					{"$push", "$hosts.src"},
				}},
				{"dst", bson.D{
					{"$push", "$hosts.dst"},
				}},
				{"max_duration", bson.D{
					{"$max", "$hosts.max_duration"},
				}},
			}},
		},
		{
			{"$project", bson.D{
				// Disable the normal _id field and use ip instead
				{"_id", 0},
				{"ip", "$_id"},
				{"local", 1},
				{"ipv4", bson.D{
					// Determines if the ip (_id) is IPv4 rather than IPv6.
					// If the ip does not contain the ':' character (IPv6 separator)
					// it is IPv4.
					{"$cond", bson.D{
						{"if", bson.D{
							{"$eq", []interface{}{
								bson.D{
									{"$indexOfCP", []interface{}{
										"$_id", ":",
									}},
								},
								-1,
							}},
						}},
						{"then", bson.D{
							{"$literal", true},
						}},
						{"else", bson.D{
							{"$literal", false},
						}},
					}},
				}},
				// Store the number of times the IP was the src of a connection
				{"count_src", bson.D{
					{"$size", "$src"},
				}},
				// Store the number of times the IP was the dst of a connection
				{"count_dst", bson.D{
					{"$size", "$dst"},
				}},
				{"max_duration", 1},
			}},
		},
		// Instead of sending this output directly to a new collection,
		// we need to iterate in order to convert IPv4 strings to binary
	}

	var queryRes struct {
		ID          bson.ObjectId `bson:"_id,omitempty"`
		IP          string        `bson:"ip"`
		Local       bool          `bson:"local"`
		IPv4        bool          `bson:"ipv4"`
		CountSrc    int32         `bson:"count_src"`
		CountDst    int32         `bson:"count_dst"`
		MaxDuration float32       `bson:"max_duration"`
	}

	// execute query
	uconnIter := res.DB.AggregateCollection(sourceCollection, session, uconnsFindQuery)

	var output []*structure.Host
	// iterate over results and convert IPv4 string to binary representation
	for uconnIter.Next(&queryRes) {

		entry := &structure.Host{
			IP:          queryRes.IP,
			Local:       queryRes.Local,
			IPv4:        queryRes.IPv4,
			CountSrc:    queryRes.CountSrc,
			CountDst:    queryRes.CountDst,
			MaxDuration: queryRes.MaxDuration,
		}

		ip := net.ParseIP(queryRes.IP)
		if queryRes.IPv4 {
			entry.IPv4Binary = ipv4ToBinary(ip)
		} // else {       // *** Note: for future ipv6 support *** //
		// 	entry.IPv6Binary = ipv6ToBinary(ip)
		// }

		if queryRes.Local {

			// add highest beacon conncount and score
			var beaconRes struct {
				ConnectionCount int     `bson:"connection_count"`
				Score           float64 `bson:"score"`
			}
			err1 := session.DB(res.DB.GetSelectedDB()).C(conf.T.Beacon.BeaconTable).Find(bson.M{"src": queryRes.IP}).Sort("-score").Limit(1).One(&beaconRes)
			if err1 == nil {
				entry.MaxBeaconScore = beaconRes.Score
				entry.MaxBeaconConnCount = beaconRes.ConnectionCount
			}

			// Count how many times the host made a TXT query
			txtCount, err2 := session.DB(res.DB.GetSelectedDB()).C(conf.T.Structure.DNSTable).
				Find(bson.M{
					"$and": []bson.M{
						bson.M{"id_orig_h": queryRes.IP},
						bson.M{"qtype_name": "TXT"},
					}}).Count()

			if err2 == nil {
				entry.TxtQueryCount = txtCount
			}

		}

		output = append(output, entry)

	}

	hostWriter(output, res.DB, conf, targetCollection)
}

// hostWriter inserts host entries into the database in bulk using buffer
func hostWriter(output []*structure.Host, resDB *database.DB, resConf *config.Config, targetCollection string) {

	// buffer length controls amount of ram used while exporting
	bufferLen := resConf.S.Bro.ImportBuffer

	// Create a buffer to hold a portion of the results
	buffer := make([]interface{}, 0, bufferLen)
	// while we can still iterate through the data add to the buffer
	for _, data := range output {

		// if the buffer is full, send to the remote database and clear buffer
		if len(buffer) == bufferLen {

			err := bulkWriteHosts(buffer, resDB, resConf, targetCollection)

			if err != nil && err.Error() != "invalid BulkError instance: no errors" {
				fmt.Println(buffer)
				fmt.Println("write error 2", err)
			}

			buffer = buffer[:0]

		}

		buffer = append(buffer, data)
	}

	//send any data left in the buffer to the remote database
	err := bulkWriteHosts(buffer, resDB, resConf, targetCollection)
	if err != nil && err.Error() != "invalid BulkError instance: no errors" {
		fmt.Println(buffer)
		fmt.Println("write error 2", err)
	}

}

// bulkWriteHosts uses MongoDB's Bulk API to insert entries into a collection.
// It also allows out of order writes to speed things up. This is the fastest
// way we know of to get data into the database.
func bulkWriteHosts(buffer []interface{}, resDB *database.DB, resConf *config.Config, targetCollection string) error {
	ssn := resDB.Session.Copy()
	defer ssn.Close()

	// set up for bulk write to database
	bulk := ssn.DB(resDB.GetSelectedDB()).C(targetCollection).Bulk()
	// writes can be sent out of order
	bulk.Unordered()
	// inserts everything in the buffer into the bulk write object as a list
	// of single interfaces
	bulk.Insert(buffer...)

	// runs all queued operations
	_, err := bulk.Run()

	return err
}
