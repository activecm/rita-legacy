package blacklist

import (
	"fmt"
	"io"
	"net/http"
	"os"

	bl "github.com/activecm/rita-bl"
	blDB "github.com/activecm/rita-bl/database"
	"github.com/activecm/rita-bl/list"
	"github.com/activecm/rita-bl/sources/lists"
	"github.com/activecm/rita-bl/sources/rpc"
	"github.com/activecm/rita/config"
	"github.com/activecm/rita/resources"
	log "github.com/sirupsen/logrus"
	mgo "github.com/globalsign/mgo"
)

type resultsChan chan map[string][]blDB.BlacklistResult

const ritaBLBufferSize = 1000

//BuildBlacklistedCollections builds the blacklisted sources,
//blacklisted destinations, blacklist hostnames, and blacklisted urls
//collections
func BuildBlacklistedCollections(res *resources.Resources) {
	//capture the current value for the error closure below
	currentDB := res.DB.GetSelectedDB()
	var err error
	var blDatabase blDB.Handle
	if res.Config.S.MongoDB.TLS.Enabled {
		blDatabase, err = blDB.NewSecureMongoDB(
			res.Config.S.MongoDB.ConnectionString,
			res.Config.R.MongoDB.AuthMechanismParsed,
			"rita-bl",
			res.Config.R.MongoDB.TLS.TLSConfig,
		)
	} else {
		blDatabase, err = blDB.NewMongoDB(
			res.Config.S.MongoDB.ConnectionString,
			res.Config.R.MongoDB.AuthMechanismParsed,
			"rita-bl",
		)
	}
	if err != nil {
		res.Log.Error(err)
		fmt.Println("\t[!] Could not connect to blacklist database")
		return
	}

	//set up rita-blacklist
	ritaBL := bl.NewBlacklist(
		blDatabase,
		func(err error) { //error handler
			res.Log.WithFields(log.Fields{
				"db": currentDB,
			}).Error(err)
		},
	)

	//set up the lists to check against
	ritaBL.SetLists(buildBlacklists(res.Config)...)

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
		res.Config.T.Structure.UniqueConnTable,
		ssn,
		uniqueSourcesAggregation,
	)
	uniqueDestIter := res.DB.AggregateCollection(
		res.Config.T.Structure.UniqueConnTable,
		ssn,
		uniqueDestAggregation,
	)
	hostnamesIter := ssn.DB(currentDB).C(res.Config.T.DNS.HostnamesTable).
		Find(nil).Iter()
	urlIter := ssn.DB(currentDB).C(res.Config.T.Urls.UrlsTable).
		Find(nil).Iter()

	//create the collections
	sourceIPs := res.Config.T.Blacklisted.SourceIPsTable
	destIPs := res.Config.T.Blacklisted.DestIPsTable
	hostnames := res.Config.T.Blacklisted.HostnamesTable
	urls := res.Config.T.Blacklisted.UrlsTable

	collections := []string{sourceIPs, destIPs, hostnames, urls}
	for _, collection := range collections {
		ssn.DB(currentDB).C(collection).Create(&mgo.CollectionInfo{
			DisableIdIndex: true,
		})
	}

	//create the data
	//TODO: refactor these into modules
	buildBlacklistedIPs(uniqueSourceIter, res, ritaBL, ritaBLBufferSize, true)

	buildBlacklistedIPs(uniqueDestIter, res, ritaBL, ritaBLBufferSize, false)

	buildBlacklistedHostnames(hostnamesIter, res, ritaBL, ritaBLBufferSize)

	buildBlacklistedURLs(urlIter, res, ritaBL, ritaBLBufferSize, "http://")

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
func buildBlacklists(conf *config.Config) []list.List {
	//build up the lists
	var blacklists []list.List
	//use prebuilt lists
	if conf.S.Blacklisted.UseIPms {
		blacklists = append(blacklists, lists.NewMyIPmsList())
	}
	if conf.S.Blacklisted.UseDNSBH {
		blacklists = append(blacklists, lists.NewDNSBHList())
	}
	if conf.S.Blacklisted.UseMDL {
		blacklists = append(blacklists, lists.NewMdlList())
	}
	//use custom lists
	ipLists := buildCustomBlacklists(
		list.BlacklistedIPType,
		conf.S.Blacklisted.IPBlacklists,
	)

	hostLists := buildCustomBlacklists(
		list.BlacklistedHostnameType,
		conf.S.Blacklisted.HostnameBlacklists,
	)

	urlLists := buildCustomBlacklists(
		list.BlacklistedURLType,
		conf.S.Blacklisted.URLBlacklists,
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
			86400, // default cache time of 1 day
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
func buildBlacklistRPCS(res *resources.Resources) []rpc.RPC {
	var rpcs []rpc.RPC
	//set up google url checker
	if len(res.Config.S.Blacklisted.SafeBrowsing.APIKey) > 0 &&
		len(res.Config.S.Blacklisted.SafeBrowsing.Database) > 0 {
		googleRPC, err := rpc.NewGoogleSafeBrowsingURLsRPC(
			res.Config.S.Blacklisted.SafeBrowsing.APIKey,
			res.Config.S.Blacklisted.SafeBrowsing.Database,
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
