package structure

import (
	"encoding/binary"
	"net"

	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"
	structureTypes "github.com/ocmdev/rita/datatypes/structure"
	log "github.com/sirupsen/logrus"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// BuildHostsCollection builds the 'host' collection for this timeframe.
// Runs via mongodb aggregation. Sourced from the 'conn' table.
func BuildHostsCollection(res *database.Resources) {
	// Create the aggregate command
	sourceCollectionName,
		newCollectionName,
		newCollectionKeys,
		pipeline := getHosts(res.Config)

	// Aggregate it!
	errorCheck := res.DB.CreateCollection(newCollectionName, false, newCollectionKeys)
	if errorCheck != nil {
		res.Log.Error("Failed: ", newCollectionName, errorCheck)
		return
	}

	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	res.DB.AggregateCollection(sourceCollectionName, ssn, pipeline)
	setIPv4Binary(res.DB.GetSelectedDB(), newCollectionName, ssn, res.Log)
	setIPv6Binary(res.DB.GetSelectedDB(), newCollectionName, ssn, res.Log)
}

//getHosts aggregates the individual hosts from the conn collection and
//labels them as private or public as well as ipv4 or ipv6. The aggregation
//includes padding for a binary encoding of the ip address.
func getHosts(conf *config.Config) (string, string, []mgo.Index, []bson.D) {
	// Name of source collection which will be aggregated into the new collection
	sourceCollectionName := conf.T.Structure.ConnTable

	// Name of the new collection
	newCollectionName := conf.T.Structure.HostTable

	// Desired indeces
	keys := []mgo.Index{
		{Key: []string{"ip"}, Unique: true},
		{Key: []string{"local"}},
		{Key: []string{"ipv4"}},
	}

	// Aggregation script
	// nolint: vet
	pipeline := []bson.D{
		{
			{"$project", bson.D{
				{"hosts", []interface{}{
					bson.D{
						{"ip", "$id_origin_h"},
						{"local", "$local_orig"},
					},
					bson.D{
						{"ip", "$id_resp_h"},
						{"local", "$local_resp"},
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
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"ip", "$_id"},
				{"local", 1},
				{"ipv4", bson.D{
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
			}},
		},
		{
			{"$out", newCollectionName},
		},
	}

	return sourceCollectionName, newCollectionName, keys, pipeline
}

//setIPv4Binary sets the binary data for the ipv4 addresses in the dataset
func setIPv4Binary(selectedDB string, collectionName string,
	session *mgo.Session, logger *log.Logger) {
	coll := session.DB(selectedDB).C(collectionName)

	i := 0

	var host structureTypes.Host
	iter := coll.Find(bson.D{{"ipv4", true}}).Snapshot().Iter() //nolint: vet

	bulkUpdate := coll.Bulk()

	for iter.Next(&host) {
		//1000 is the most a MongoDB bulk update operation can handle
		if i == 1000 {
			bulkUpdate.Unordered()
			_, err := bulkUpdate.Run()
			if err != nil {
				logger.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("Unable to write binary representation of IP addresses")
			}

			bulkUpdate = coll.Bulk()
			i = 0
		}

		ipv4 := net.ParseIP(host.Ip)
		ipv4Binary := uint64(binary.BigEndian.Uint32(ipv4[12:16]))

		//nolint: vet
		bulkUpdate.Update(
			bson.D{
				{"_id", host.ID},
			},
			bson.D{
				{"$set", bson.D{
					{"ipv4_binary", ipv4Binary},
				}},
			},
		)
		i++
	}

	//guaranteed to be at least one in the array
	bulkUpdate.Unordered()
	_, err := bulkUpdate.Run()
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to write binary representation of IP addresses")
	}
}

//setIPv6Binary sets the binary data for the ipv6 addresses in the dataset
func setIPv6Binary(selectedDB string, collectionName string,
	session *mgo.Session, logger *log.Logger) {
	coll := session.DB(selectedDB).C(collectionName)

	i := 0

	var host structureTypes.Host
	iter := coll.Find(bson.D{{"ipv4", false}}).Snapshot().Iter() //nolint: vet

	bulkUpdate := coll.Bulk()

	for iter.Next(&host) {
		//1000 is the most a MongoDB bulk update operation can handle
		if i == 1000 {
			bulkUpdate.Unordered()
			_, err := bulkUpdate.Run()
			if err != nil {
				logger.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("Unable to write binary representation of IP addresses")
			}

			bulkUpdate = coll.Bulk()
			i = 0
		}

		ipv6 := net.ParseIP(host.Ip)
		ipv6Binary1 := uint64(binary.BigEndian.Uint32(ipv6[0:4]))
		ipv6Binary2 := uint64(binary.BigEndian.Uint32(ipv6[4:8]))
		ipv6Binary3 := uint64(binary.BigEndian.Uint32(ipv6[8:12]))
		ipv6Binary4 := uint64(binary.BigEndian.Uint32(ipv6[12:16]))

		//nolint: vet
		bulkUpdate.Update(
			bson.D{
				{"_id", host.ID},
			},
			bson.D{
				{"$set", bson.D{
					{"ipv6_binary", bson.D{
						{"1", ipv6Binary1},
						{"2", ipv6Binary2},
						{"3", ipv6Binary3},
						{"4", ipv6Binary4},
					}},
				}},
			},
		)

		i++
	}

	//guaranteed to be at least one in the array
	bulkUpdate.Unordered()
	_, err := bulkUpdate.Run()
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to write binary representation of IP addresses")
	}
}
