package blacklist

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/ocmdev/mgosec"
	bl "github.com/ocmdev/rita-bl"
	blDB "github.com/ocmdev/rita-bl/database"
	"github.com/ocmdev/rita-bl/list"
	"github.com/ocmdev/rita-bl/sources/lists"
	"github.com/ocmdev/rita-bl/sources/rpc"
	"github.com/ocmdev/rita/config"
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

	blDB, err := blDB.NewMongoDB(res.System.DatabaseHost, mgosec.None, "rita-bl")
	if err != nil {
		res.Log.Error(err)
		fmt.Println("\t[!] Could not connect to blacklist database")
		return
	}

	//set up rita-blacklist
	ritaBL := bl.NewBlacklist(
		blDB,
		func(err error) { //error handler
			res.Log.WithFields(log.Fields{
				"db": currentDB,
			}).Error(err)
		},
	)

	//set up the lists to check against
	ritaBL.SetLists(buildBlacklists(res.System)...)

	//set up remote calls to check against
	ritaBL.SetRPCs(buildBlacklistRPCS(res)...)

	//update the lists
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
	hostnamesIter := ssn.DB(currentDB).C(res.System.DNSConfig.HostnamesTable).
		Find(nil).Iter()
	urlIter := ssn.DB(currentDB).C(res.System.UrlsConfig.UrlsTable).
		Find(nil).Iter()

	//create the collections
	sourceIPs := res.System.BlacklistedConfig.SourceIPsTable
	destIPs := res.System.BlacklistedConfig.DestIPsTable
	hostnames := res.System.BlacklistedConfig.HostnamesTable
	urls := res.System.BlacklistedConfig.UrlsTable

	collections := []string{sourceIPs, destIPs, hostnames, urls}
	for _, collection := range collections {
		ssn.DB(currentDB).C(collection).Create(&mgo.CollectionInfo{
			DisableIdIndex: true,
		})
	}

	//create the data
	//TODO: refactor these into modules
	bufferSize := 1000
	buildBlacklistedIPs(uniqueSourceIter, res, ritaBL, bufferSize, true)

	buildBlacklistedIPs(uniqueDestIter, res, ritaBL, bufferSize, false)

	buildBlacklistedHostnames(hostnamesIter, res, ritaBL, bufferSize)

	buildBlacklistedURLs(urlIter, res, ritaBL, bufferSize, "http://")

	//index the data
	for _, collection := range collections {
		ensureBLIndexes(ssn, currentDB, collection)
	}

	ssn.DB(currentDB).C(sourceIPs).EnsureIndex(mgo.Index{
		Key: []string{"$hashed:ip"},
	})

	ssn.DB(currentDB).C(destIPs).EnsureIndex(mgo.Index{
		Key: []string{"$hashed:ip"},
	})
	ssn.DB(currentDB).C(hostnames).EnsureIndex(mgo.Index{
		Key: []string{"$hashed:hostname"},
	})
	ssn.DB(currentDB).C(urls).EnsureIndex(mgo.Index{
		Key: []string{"host", "resource"},
	})
}

//ensureBLIndexes ensures the sortable fields are indexed
//on the blacklist results
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

//buildBlacklists gathers the blacklists to check against
func buildBlacklists(system *config.SystemConfig) []list.List {
	//build up the lists
	var blacklists []list.List
	//use prebuilt lists
	if system.BlacklistedConfig.UseIPms {
		blacklists = append(blacklists, lists.NewMyIPmsList())
	}
	if system.BlacklistedConfig.UseDNSBH {
		blacklists = append(blacklists, lists.NewDNSBHList())
	}
	if system.BlacklistedConfig.UseMDL {
		blacklists = append(blacklists, lists.NewMdlList())
	}
	//use custom lists
	ipLists := buildCustomBlacklists(
		list.BlacklistedIPType,
		system.BlacklistedConfig.IPBlacklists,
	)

	hostLists := buildCustomBlacklists(
		list.BlacklistedHostnameType,
		system.BlacklistedConfig.HostnameBlacklists,
	)

	urlLists := buildCustomBlacklists(
		list.BlacklistedURLType,
		system.BlacklistedConfig.URLBlacklists,
	)
	blacklists = append(blacklists, ipLists...)
	blacklists = append(blacklists, hostLists...)
	blacklists = append(blacklists, urlLists...)
	return blacklists
}

//buildCustomBlacklists gathers a custom blacklist from a url or file path
func buildCustomBlacklists(entryType list.BlacklistedEntryType, paths []string) []list.List {
	var blacklists []list.List
	for _, path := range paths {
		newList := lists.NewLineSeperatedList(
			entryType,
			path,
			86400,
			tryOpenFileThenURL(path),
		)
		blacklists = append(blacklists, newList)
	}
	return blacklists
}

//provide a closure over path to read the file into a line separated blacklist
func tryOpenFileThenURL(path string) func() (io.ReadCloser, error) {
	return func() (io.ReadCloser, error) {
		_, err := os.Stat(path)
		if err == nil {
			file, err2 := os.Open(path)
			if err2 != nil {
				return nil, err2
			}
			return file, nil
		}
		resp, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	}
}

//buildBlacklistRPCS gathers the remote procedures to check against
func buildBlacklistRPCS(res *database.Resources) []rpc.RPC {
	var rpcs []rpc.RPC
	//set up google url checker
	if len(res.System.BlacklistedConfig.SafeBrowsing.APIKey) > 0 &&
		len(res.System.BlacklistedConfig.SafeBrowsing.Database) > 0 {
		googleRPC, err := rpc.NewGoogleSafeBrowsingURLsRPC(
			res.System.BlacklistedConfig.SafeBrowsing.APIKey,
			res.System.BlacklistedConfig.SafeBrowsing.Database,
			res.Log.Writer(),
		)
		if err == nil {
			rpcs = append(rpcs, googleRPC)
		} else {
			res.Log.Warn("could not open up google safebrowsing for blacklist checks")
		}
	}
	return rpcs
}
