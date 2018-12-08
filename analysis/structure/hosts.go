package structure

import (
	"fmt"
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

	// res.DB.AggregateCollection(sourceCollectionName, ssn, pipeline)
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
					},
					bson.D{
						{"ip", "$dst"},
						{"local", "$local_dst"},
						{"dst", true},
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
			}},
		},
		// Instead of sending this output directly to a new collection,
		// we need to iterate in order to convert IPv4 strings to binary
	}

	var queryRes struct {
		ID       bson.ObjectId `bson:"_id,omitempty"`
		IP       string        `bson:"ip"`
		Local    bool          `bson:"local"`
		IPv4     bool          `bson:"ipv4"`
		CountSrc int32         `bson:"count_src"`
		CountDst int32         `bson:"count_dst"`
	}

	// execute query
	uconnIter := session.DB(res.DB.GetSelectedDB()).
		C(sourceCollection).
		Pipe(uconnsFindQuery).Iter()

	var output []*structure.Host
	// iterate over results and convert IPv4 string to binary representation
	for uconnIter.Next(&queryRes) {

		entry := &structure.Host{
			IP:       queryRes.IP,
			Local:    queryRes.Local,
			IPv4:     queryRes.IPv4,
			CountSrc: queryRes.CountSrc,
			CountDst: queryRes.CountDst,
		}

		ip := net.ParseIP(queryRes.IP)
		if queryRes.IPv4 {
			entry.IPv4Binary = ipv4ToBinary(ip)
		} // else {       // *** Note: for future ipv6 support *** //
		// 	entry.IPv6Binary = ipv6ToBinary(ip)
		// }

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
