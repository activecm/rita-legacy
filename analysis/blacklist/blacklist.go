package blacklist

import (
	"fmt"
	"io"
	"net/http"
	"os"

	ritaBL "github.com/activecm/rita-bl"
	ritaBLdb "github.com/activecm/rita-bl/database"
	"github.com/activecm/rita-bl/list"
	"github.com/activecm/rita-bl/sources/lists"
	"github.com/activecm/rita/config"
	"github.com/activecm/rita/resources"
	log "github.com/sirupsen/logrus"
)

//BuildBlacklistedCollections builds the blacklist master reference collection
//and checks the uconn documents against that collection
func BuildBlacklistedCollections(res *resources.Resources) {
	// build the blacklist reference collection from provided blacklist sources
	// this will be the master list ips and hostnames will be checked against
	buildBlacklistReferenceCollection(res)

	// build src ip collection
	buildBlacklistedIPs(res, true)

	//build dst ip collection
	buildBlacklistedIPs(res, false)

	//build hostnames collection
	buildBlacklistedHostnames(res)

}

func buildBlacklistReferenceCollection(res *resources.Resources) {

	/***************** create new blacklist collection *********************/
	var err error
	var blDatabase ritaBLdb.Handle

	// set current dataset name
	currentDB := res.DB.GetSelectedDB()

	// User option set in yaml file, if enabled
	// RITA will verify the MongoDB certificate's hostname and validity
	// otherwise run a normal request to create a database
	if res.Config.S.MongoDB.TLS.Enabled {
		blDatabase, err = ritaBLdb.NewSecureMongoDB(
			res.Config.S.MongoDB.ConnectionString,
			res.Config.R.MongoDB.AuthMechanismParsed,
			res.Config.S.Blacklisted.BlacklistDatabase,
			res.Config.R.MongoDB.TLS.TLSConfig,
		)
	} else {
		blDatabase, err = ritaBLdb.NewMongoDB(
			res.Config.S.MongoDB.ConnectionString,
			res.Config.R.MongoDB.AuthMechanismParsed,
			res.Config.S.Blacklisted.BlacklistDatabase,
		)
	}

	if err != nil {
		res.Log.Error(err)
		fmt.Println("\t[!] Could not connect to blacklist database")
		return
	}

	// Creates new rita bl blacklist structure
	ritaBL := ritaBL.NewBlacklist(
		blDatabase,
		func(err error) { //error handler
			res.Log.WithFields(log.Fields{
				"db": currentDB,
			}).Error(err)
		},
	)

	//send blacklist source lists
	ritaBL.SetLists(getSourceLists(res.Config)...)

	//update the lists
	ritaBL.Update()

	return
}

//getSourceLists gathers the blacklists to check against
func getSourceLists(conf *config.Config) []list.List {
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

	blacklists = append(blacklists, ipLists...)
	blacklists = append(blacklists, hostLists...)

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
