package beaconsni

import (
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
)

//Results finds SNI beacons in the database greater than a given cutoffScore
func Results(res *resources.Resources, cutoffScore float64) ([]Result, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var beaconsSNI []Result

	beaconSNIQuery := bson.M{"score": bson.M{"$gt": cutoffScore}}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.BeaconSNI.BeaconSNITable).Find(beaconSNIQuery).Sort("-score").All(&beaconsSNI)
	if err != nil {
		return beaconsSNI, err
	}
	// for idx, _ := range beaconsSNI {
	// 	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.SNIConnTable).Pipe([]bson.M{
	// 		{"$match": beaconsSNI[idx].UniqueSrcFQDNPair.BSONKey()},
	// 		{"$project": bson.M{
	// 			"dst_ips": bson.M{"$concatArrays": []string{"$dat.http.dst_ips", "$dat.tls.dst_ips"}},
	// 		}},
	// 		{"$unwind": "$dst_ips"},
	// 		{"$unwind": "$dst_ips"},
	// 		{"$group": bson.M{
	// 			"_id": bson.M{
	// 				"ip":           "$dst_ips.ip",
	// 				"network_uuid": "$dst_ips.network_uuid",
	// 			},
	// 			"network_name": bson.M{"$last": "$dst_ips.network_name"},
	// 		}},
	// 		{"$project": bson.M{
	// 			"_id":          0,
	// 			"ip":           "$_id.ip",
	// 			"network_uuid": "$_id.network_uuid",
	// 			"network_name": 1,
	// 		}},
	// 	}).All(&(beaconsSNI[idx].ResolvedIPs))
	// 	if err != nil {
	// 		return beaconsSNI, err
	// 	}
	// }

	return beaconsSNI, err
}
