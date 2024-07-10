package beaconproxy

import (
	"github.com/activecm/rita-legacy/resources"
	"github.com/globalsign/mgo/bson"
)

// Results finds beacons FQDN in the database greater than a given cutoffScore
func Results(res *resources.Resources, cutoffScore float64) ([]Result, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var beaconsProxy []Result

	BeaconProxyQuery := bson.M{"score": bson.M{"$gt": cutoffScore}}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.BeaconProxy.BeaconProxyTable).Find(BeaconProxyQuery).Sort("-score").All(&beaconsProxy)

	return beaconsProxy, err
}
