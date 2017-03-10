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

func buildExplodedDNSVistedCounts(res *database.Resources) {
	res.DB.MapReduceCollection(
		res.System.StructureConfig.DnsTable,
		mgo.MapReduce{
			Map: `function() {
              var dots = [];
              //find all subdomain seperators
              for (i = 0; i < this.query.length; i++) {
                  if (this.query[i] == '.') {
                      dots.push(i);
                  }
              }
              for (i = 0; i < dots.length; i++) {
                  emit(this.query.substring(dots[i] + 1), 1);
              }
            }`,
			Reduce: `function(subdomain, countArr) {
                return Array.sum(countArr);
               }`,
			Out: bson.M{"replace": tempVistedCountCollName},
		},
	)
}

func buildExplodedDNSUniqSubdomains(res *database.Resources) {

}

func zipExplodedDNSResults(res *database.Resources) {

}
