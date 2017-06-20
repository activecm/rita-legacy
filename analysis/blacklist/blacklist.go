package blacklist

import (
	bl "github.com/ocmdev/rita-blacklist2"
	blDB "github.com/ocmdev/rita-blacklist2/database"
	"github.com/ocmdev/rita-blacklist2/sources/lists"
	"github.com/ocmdev/rita-blacklist2/sources/rpc"
	"github.com/ocmdev/rita/database"
)

type resultsChan chan map[string][]blDB.BlacklistResult

//BuildBlacklistedCollections builds the blacklisted sources,
//blacklisted destinations, blacklist hostnames, and blacklisted urls
//collections
func BuildBlacklistedCollections(res *database.Resources) {
	//set up rita-blacklist
	ritaBL := bl.NewBlacklist(
		blDB.NewMongoDB,
		res.System.DatabaseHost,
		"rita-blacklist2",
		func(err error) {
			res.Log.Error(err)
		},
	)

	googleRPC, err := rpc.NewGoogleSafeBrowsingURLsRPC(
		res.System.SafeBrowsing.APIKey,
		res.System.SafeBrowsing.Database,
		res.Log.Writer(),
	)
	if err == nil {
		ritaBL.SetRPCs(googleRPC)
	} else {
		res.Log.Error("could not open up google safebrowsing for blacklist checks")
	}

	ritaBL.SetLists(lists.NewMyIPmsList(), lists.NewMdlList())
	ritaBL.Update()

	//get our data sources
	ssn := res.DB.Session.Copy()
	defer ssn.Close()
	uniqueSourcesAggregation := getUniqueIPFromUconnPipeline("src")
	uniqueDestAggregation := getUniqueIPFromUconnPipeline("dst")
	uniqueSourceIter := res.DB.AggregateCollection(
		res.System.StructureConfig.UniqueConnTable,
		ssn,
		uniqueSourcesAggregation,
	)
	uniqueDestIter := res.DB.AggregateCollection(
		res.System.StructureConfig.UniqueConnTable,
		ssn,
		uniqueDestAggregation,
	)
	hostnamesIter := ssn.DB(res.DB.GetSelectedDB()).C(
		res.System.DNSConfig.HostnamesTable,
	).Find(nil).Iter()
	urlIter := ssn.DB(res.DB.GetSelectedDB()).C(
		res.System.UrlsConfig.UrlsTable,
	).Find(nil).Iter()

	bufferSize := 1000

	buildBlacklistedSourceIPs(
		uniqueSourceIter, ssn, ritaBL,
		"blSourceIPs", bufferSize,
	)

	buildBlacklistedDestIPs(
		uniqueDestIter, ssn, ritaBL,
		"blDestIPs", bufferSize,
	)

	buildBlacklistedHostnames(
		hostnamesIter, ssn, ritaBL,
		"blHostnames", bufferSize,
	)

	buildBlacklistedURLs(
		urlIter, ssn, ritaBL,
		"blURLs", bufferSize,
	)

}
