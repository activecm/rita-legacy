package dns

import (
	"github.com/ocmdev/rita/database"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var tempVistedCountCollName string = "__temp_ExplodedDNSVistedCounts"
var tempUniqSubdomainCollName string = "__temp_UniqSubdomains"

// BuildExplodedDNSCollection splits domain names into sub-domains
// and performs analysis
func BuildExplodedDNSCollection(res *database.Resources) {
	buildExplodedDNSVistedCounts(res)
	buildExplodedDNSUniqSubdomains(res)
	zipExplodedDNSResults(res)
}

// buildExplodedDNSVistedCounts uses the map reduce job to count how many
// times each super domain was visited
func buildExplodedDNSVistedCounts(res *database.Resources) {
	res.DB.MapReduceCollection(
		res.System.StructureConfig.DnsTable,
		mgo.MapReduce{
			Map:    getExplodedDNSMapper("query"),
			Reduce: getExplodedDNSReducer(),
			Out:    bson.M{"replace": tempVistedCountCollName},
		},
	)
}

// buildExplodedDNSUniqSubdomains uses the map reduce job to count how many
// unique subdomains exist for a given super domain
func buildExplodedDNSUniqSubdomains(res *database.Resources) {
	res.DB.MapReduceCollection(
		tempVistedCountCollName,
		mgo.MapReduce{
			Map:    getExplodedDNSMapper("_id"),
			Reduce: getExplodedDNSReducer(),
			Out:    bson.M{"replace": tempUniqSubdomainCollName},
		},
	)
}

func zipExplodedDNSResults(res *database.Resources) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()
	res.DB.CreateCollection(res.System.DnsConfig.ExplodedDnsTable, []string{"domain"})
	res.DB.AggregateCollection(tempVistedCountCollName, ssn,
		[]bson.D{
			{
				{"$lookup", bson.D{
					{"from", tempUniqSubdomainCollName},
					{"localField", "_id"},
					{"foreignField", "_id"},
					{"as", "subdomains"},
				}},
			},
			{
				{"$unwind", "$subdomains"},
			},
			{
				{"$project", bson.D{
					{"_id", 0},
					{"domain", "$_id"},
					{"visited", "$value"},
					{"subdomains", "$subdomains.value"},
				}},
			},
			{
				{"$out", res.System.DnsConfig.ExplodedDnsTable},
			},
		},
	)
}

//Inserting a variable into a javascript function what could go wrong

// getExplodedDNSMapper creates on O(N) map reduce job which
// grabs all of the superdomains from a fqdn e.g. maps.google.com produces
// maps.google.com, google.com, and com
func getExplodedDNSMapper(nameField string) string {
	return `function() {
		var dots = [];
		var domain = this.` + nameField + `.toLowerCase();
		//find all subdomain separators
		for (i = 0; i < domain.length; i++) {
				if (domain[i] == '.') {
						dots.push(i);
				}
		}
		//emit all of the "super domains"
		emit(domain, 1);
		for (i = 0; i < dots.length; i++) {
				emit(domain.substring(dots[i] + 1), 1);
		}
	}`
}

func getExplodedDNSReducer() string {
	return `function(subdomain, countArr) {
						return Array.sum(countArr);
					}`
}
