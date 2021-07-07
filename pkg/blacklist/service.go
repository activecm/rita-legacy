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
	"github.com/activecm/rita/database"
	log "github.com/sirupsen/logrus"
)

// type Service interface {
// 	BuildBlacklistedCollections(res *resources.Resources)
// }

// func NewService() *Service {
// 	return &Service{}
// }

//BuildBlacklistedCollections builds the blacklist master reference collection
//and checks the uconn documents against that collection
func BuildBlacklistedCollections(db *database.DB, conf *config.Config, logger *log.Logger) {
	// build the blacklist reference collection from provided blacklist sources
	// this will be the master list ips and hostnames will be checked against
	buildBlacklistReferenceCollection(db, conf, logger)

	// // build src ip collection
	// buildBlacklistedIPs(res, true)

	// //build dst ip collection
	// buildBlacklistedIPs(res, false)

	// //build hostnames collection
	// buildBlacklistedHostnames(res)

}

func buildBlacklistReferenceCollection(db *database.DB, conf *config.Config, logger *log.Logger) {

	/***************** create new blacklist collection *********************/
	var err error
	var blDatabase ritaBLdb.Handle

	// set current dataset name
	currentDB := db.GetSelectedDB()

	// User option set in yaml file, if enabled
	// RITA will verify the MongoDB certificate's hostname and validity
	// otherwise run a normal request to create a database
	if conf.S.MongoDB.TLS.Enabled {
		blDatabase, err = ritaBLdb.NewSecureMongoDB(
			conf.S.MongoDB.ConnectionString,
			conf.R.MongoDB.AuthMechanismParsed,
			conf.S.Blacklisted.BlacklistDatabase,
			conf.R.MongoDB.TLS.TLSConfig,
		)
	} else {
		blDatabase, err = ritaBLdb.NewMongoDB(
			conf.S.MongoDB.ConnectionString,
			conf.R.MongoDB.AuthMechanismParsed,
			conf.S.Blacklisted.BlacklistDatabase,
		)
	}

	if err != nil {
		logger.Error(err)
		fmt.Println("\t[!] Could not connect to blacklist database")
		return
	}

	// Creates new rita bl blacklist structure
	ritaBL := ritaBL.NewBlacklist(
		blDatabase,
		func(err error) { //error handler
			logger.WithFields(log.Fields{
				"db": currentDB,
			}).Error(err)
		},
	)

	//send blacklist source lists
	ritaBL.SetLists(getSourceLists(conf)...)

	//update the lists
	ritaBL.Update()

}

//getSourceLists gathers the blacklists to check against
func getSourceLists(conf *config.Config) []list.List {
	//build up the lists
	var blacklists []list.List
	//use prebuilt lists
	if conf.S.Blacklisted.UseDNSBH {
		blacklists = append(blacklists, lists.NewDNSBHList())
	}
	if conf.S.Blacklisted.UseFeodo {
		blacklists = append(blacklists, lists.NewFeodoList())
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
		newList := lists.NewLineSeparatedList(
			entryType,
			path,
			0, // Always reload the data
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
