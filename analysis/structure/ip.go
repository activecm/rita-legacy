package structure

import (
	"encoding/binary"
	"net"

	structureTypes "github.com/activecm/rita/datatypes/structure"
	"github.com/activecm/rita/resources"
	log "github.com/sirupsen/logrus"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//BuildIPv4Collection generates binary representations of the IPv4 addresses in the
//dataset for use in subnetting, address selection, etc.
func BuildIPv4Collection(res *resources.Resources) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	errorCheck := res.DB.CreateCollection(
		res.Config.T.Structure.IPv4Table,
		[]mgo.Index{
			{Key: []string{"ip"}, Unique: true},
			{Key: []string{"ipv4_binary"}},
		},
	)
	if errorCheck != nil {
		res.Log.Error("Failed: ", res.Config.T.Structure.IPv4Table, errorCheck)
		return
	}

	buildIPv4Binary(
		res.DB.GetSelectedDB(),
		res.Config.T.Structure.HostTable,
		res.Config.T.Structure.IPv4Table,
		ssn,
		res.Log,
	)
}

//BuildIPv6Collection generates binary representations of the IPv6 addresses in the
//dataset for use in subnetting, address selection, etc.
func BuildIPv6Collection(res *resources.Resources) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	errorCheck := res.DB.CreateCollection(
		res.Config.T.Structure.IPv6Table,
		[]mgo.Index{
			{Key: []string{"ip"}, Unique: true},
			{Key: []string{"ipv6_binary.1"}},
			{Key: []string{"ipv6_binary.2"}},
			{Key: []string{"ipv6_binary.3"}},
			{Key: []string{"ipv6_binary.4"}},
		},
	)
	if errorCheck != nil {
		res.Log.Error("Failed: ", res.Config.T.Structure.IPv6Table, errorCheck)
		return
	}

	buildIPv6Binary(
		res.DB.GetSelectedDB(),
		res.Config.T.Structure.HostTable,
		res.Config.T.Structure.IPv6Table,
		ssn,
		res.Log,
	)
}

//buildIPv4Binary sets the binary data for the ipv4 addresses in the dataset
func buildIPv4Binary(selectedDB, hostCollection, destCollection string,
	session *mgo.Session, logger *log.Logger) {
	srcColl := session.DB(selectedDB).C(hostCollection)
	destColl := session.DB(selectedDB).C(destCollection)
	i := 0

	var host structureTypes.Host
	iter := srcColl.Find(bson.D{{"ipv4", true}}).Iter() //nolint: vet

	bulkUpdate := destColl.Bulk()

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

			bulkUpdate = destColl.Bulk()
			i = 0
		}

		ipv4 := net.ParseIP(host.IP)
		ipv4Struct := structureTypes.IPv4Binary{
			IP:         host.IP,
			IPv4Binary: ipv4ToBinary(ipv4),
		}
		bulkUpdate.Insert(ipv4Struct)

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

func ipv4ToBinary(ipv4 net.IP) int64 {
	return int64(binary.BigEndian.Uint32(ipv4[12:16]))
}

//buildIPv6Binary sets the binary data for the ipv6 addresses in the dataset
func buildIPv6Binary(selectedDB, hostCollection, destCollection string,
	session *mgo.Session, logger *log.Logger) {
	srcColl := session.DB(selectedDB).C(hostCollection)
	destColl := session.DB(selectedDB).C(destCollection)
	i := 0

	var host structureTypes.Host
	iter := srcColl.Find(bson.D{{"ipv4", false}}).Iter() //nolint: vet

	bulkUpdate := destColl.Bulk()

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

			bulkUpdate = destColl.Bulk()
			i = 0
		}

		ipv6 := net.ParseIP(host.IP)
		ipv6Binary1, ipv6Binary2, ipv6Binary3, ipv6Binary4 := ipv6ToBinary(ipv6)
		ipv6Struct := structureTypes.IPv6Binary{
			IP: host.IP,
			IPv6Binary: structureTypes.IPv6Integers{
				I1: ipv6Binary1,
				I2: ipv6Binary2,
				I3: ipv6Binary3,
				I4: ipv6Binary4,
			},
		}
		bulkUpdate.Insert(ipv6Struct)

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

func ipv6ToBinary(ipv6 net.IP) (int64, int64, int64, int64) {
	ipv6Binary1 := int64(binary.BigEndian.Uint32(ipv6[0:4]))
	ipv6Binary2 := int64(binary.BigEndian.Uint32(ipv6[4:8]))
	ipv6Binary3 := int64(binary.BigEndian.Uint32(ipv6[8:12]))
	ipv6Binary4 := int64(binary.BigEndian.Uint32(ipv6[12:16]))
	return ipv6Binary1, ipv6Binary2, ipv6Binary3, ipv6Binary4
}
