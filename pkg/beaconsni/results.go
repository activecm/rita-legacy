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

	return beaconsSNI, err
}
