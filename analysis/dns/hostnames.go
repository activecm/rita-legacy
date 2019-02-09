package dns

import (
	dnsTypes "github.com/activecm/rita/datatypes/dns"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
)

// GetIPsFromHost uses the hostnames table to do a cached whois query
func GetIPsFromHost(res *resources.Resources, host string) []string {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	hostnames := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.HostnamesTable)

	var destIPs dnsTypes.Hostname
	hostnames.Find(bson.M{"host": host}).One(&destIPs)

	return destIPs.IPs
}
