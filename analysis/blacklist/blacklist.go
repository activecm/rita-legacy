package blacklist

import (
	bl "github.com/ocmdev/rita-blacklist2"
	blDB "github.com/ocmdev/rita-blacklist2/database"
	"github.com/ocmdev/rita-blacklist2/sources/lists"
	"github.com/ocmdev/rita-blacklist2/sources/rpc"
	"github.com/ocmdev/rita/database"
	log "github.com/sirupsen/logrus"
	mgo "gopkg.in/mgo.v2"
)

type resultsChan chan map[string][]blDB.BlacklistResult

//BuildBlacklistedCollections builds the blacklisted sources,
//blacklisted destinations, blacklist hostnames, and blacklisted urls
//collections
func BuildBlacklistedCollections(res *database.Resources) {
	//capture the current value for the error closure below
	currentDB := res.DB.GetSelectedDB()

	//set up rita-blacklist
	ritaBL := bl.NewBlacklist(
		blDB.NewMongoDB,         //Use MongoDB for data storage
		res.System.DatabaseHost, //Use the DatabaseHost as the connection
		"rita-blacklist2",       //database
		func(err error) { //error handler
			res.Log.WithFields(log.Fields{
				"db": currentDB,
			}).Error(err)
		},
	)

	//set up google url checker
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

	//set up ritaBL to pull from myIP.ms and MDL
	ritaBL.SetLists(
		lists.NewMyIPmsList(),
		lists.NewMdlList(),
	)
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
	collections := []string{"bl-sourceIPs", "bl-destIPs", "bl-hostnames", "bl-urls"}
	for _, collection := range collections {
		ssn.DB(currentDB).C(collection).Create(&mgo.CollectionInfo{
			DisableIdIndex: true,
		})
	}
	//TODO: refactor these into modules
	buildBlacklistedIPs(uniqueSourceIter, res, ritaBL, bufferSize, true)

	buildBlacklistedIPs(uniqueDestIter, res, ritaBL, bufferSize, false)

	buildBlacklistedHostnames(hostnamesIter, res, ritaBL, bufferSize)

	buildBlacklistedURLs(urlIter, res, ritaBL, bufferSize, "http://")

	ensureBLIndexes(ssn, currentDB, "bl-sourceIPs")
	ensureBLIndexes(ssn, currentDB, "bl-destIPs")
	ensureBLIndexes(ssn, currentDB, "bl-hostnames")
	ensureBLIndexes(ssn, currentDB, "bl-urls")

	ssn.DB(currentDB).C("bl-sourceIPs").EnsureIndex(mgo.Index{
		Key: []string{"$hashed:ip"},
	})

	ssn.DB(currentDB).C("bl-destIPs").EnsureIndex(mgo.Index{
		Key: []string{"$hashed:ip"},
	})
	ssn.DB(currentDB).C("bl-hostnames").EnsureIndex(mgo.Index{
		Key: []string{"$hashed:hostname"},
	})
	ssn.DB(currentDB).C("bl-urls").EnsureIndex(mgo.Index{
		Key: []string{"host", "resource"},
	})

}

func ensureBLIndexes(ssn *mgo.Session, currentDB, collName string) {
	ssn.DB(currentDB).C(collName).EnsureIndex(mgo.Index{
		Key: []string{"conn"},
	})
	ssn.DB(currentDB).C(collName).EnsureIndex(mgo.Index{
		Key: []string{"uconn"},
	})
	ssn.DB(currentDB).C(collName).EnsureIndex(mgo.Index{
		Key: []string{"total_bytes"},
	})
}
