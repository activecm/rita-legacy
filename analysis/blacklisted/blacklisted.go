package blacklisted

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	blacklist "github.com/ocmdev/rita-blacklist"
	"github.com/ocmdev/rita/analysis/dns"
	"github.com/ocmdev/rita/analysis/structure"
	"github.com/ocmdev/rita/database"

	"github.com/google/safebrowsing"

	"github.com/ocmdev/rita/util"

	"github.com/ocmdev/rita/datatypes/blacklisted"
	datatype_structure "github.com/ocmdev/rita/datatypes/structure"

	log "github.com/sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type (
	// Blacklisted provides a handle for the blacklist module
	Blacklisted struct {
		db             string                    // database name (customer)
		batchSize      int                       // BatchSize
		prefetch       float64                   // Prefetch
		resources      *database.Resources       // resources
		log            *log.Logger               // logger
		channelSize    int                       // channel size
		threadCount    int                       // Thread count
		blacklistTable string                    // Name of blacklist table
		safeBrowser    *safebrowsing.SafeBrowser // Google safebrowsing api
		ritaBL         *blacklist.BlackList      // Blacklisted host database
	}

	// URLShort is a shortened version of the URL datatype that only accounts
	// for the IP and url (hostname)
	URLShort struct {
		URL string `bson:"host"`
	}
)

//SetBlacklistSources finds all of the sources which contacted
//the hosts on the blacklist
func SetBlacklistSources(res *database.Resources, result *blacklisted.Blacklist) {
	if result.IsURL {
		for _, destIP := range dns.GetIPsFromHost(res, result.Host) {
			result.Sources = append(result.Sources, structure.GetConnSourcesFromDest(res, destIP)...)
		}
	} else {
		result.Sources = structure.GetConnSourcesFromDest(res, result.Host)
	}
}

//BuildBlacklistedCollection runs the hosts in the dataset against rita-blacklist
func BuildBlacklistedCollection(res *database.Resources) {
	collectionName := res.System.BlacklistedConfig.BlacklistTable
	//this wil go away in the new blacklist update
	collectionKeys := []mgo.Index{
		{Key: []string{"bl_hash"}},
		{Key: []string{"host"}},
	}
	err := res.DB.CreateCollection(collectionName, false, collectionKeys)
	if err != nil {
		res.Log.Error("Failed: ", collectionName, err.Error())
		return
	}
	newBlacklisted(res).run()
}

// New will create a new blacklisted module
func newBlacklisted(res *database.Resources) *Blacklisted {

	ret := Blacklisted{
		db:             res.DB.GetSelectedDB(),
		batchSize:      res.System.BatchSize,
		prefetch:       res.System.Prefetch,
		resources:      res,
		log:            res.Log,
		channelSize:    res.System.BlacklistedConfig.ChannelSize,
		threadCount:    res.System.BlacklistedConfig.ThreadCount,
		blacklistTable: res.System.BlacklistedConfig.BlacklistTable,
	}

	// Check if the config file contains a safe browsing key
	if len(res.System.SafeBrowsing.APIKey) > 0 {
		// Initialize the hook to google's safebrowsing api.
		sbConfig := safebrowsing.Config{
			APIKey: res.System.SafeBrowsing.APIKey,
			DBPath: res.System.SafeBrowsing.Database,
			Logger: res.Log.Writer(),
		}
		sb, err := safebrowsing.NewSafeBrowser(sbConfig)
		if err != nil {
			res.Log.WithField("Error", err).Error("Error opening safe browser API")
		} else {
			ret.safeBrowser = sb
		}
	}

	// Initialize a rita-blacklist instance. Opens a database connection
	// to the blacklist database. This will cause an update if the list is out
	// of date.
	ritabl := blacklist.NewBlackList()
	hostport := strings.Split(res.System.DatabaseHost, ":")
	if len(hostport) > 1 {
		port, err := strconv.Atoi(hostport[1])
		if err == nil {
			ritabl.Init(hostport[0], port, res.System.BlacklistedConfig.BlacklistDatabase)
			ret.ritaBL = ritabl
		} else {
			res.Log.WithField("Error", err).Error("Error opening rita-blacklist hook")
		}
	}
	return &ret
}

// Run runs the module
func (b *Blacklisted) run() {
	ipssn := b.resources.DB.Session.Copy()
	defer ipssn.Close()
	urlssn := b.resources.DB.Session.Copy()
	defer urlssn.Close()

	// build up cursors
	ipcur := ipssn.DB(b.db).C(b.resources.System.StructureConfig.HostTable)
	urlcur := urlssn.DB(b.db).C(b.resources.System.DNSConfig.HostnamesTable)

	ipaddrs := make(chan string, b.channelSize)
	urls := make(chan URLShort, b.channelSize)
	cash := util.NewCache()
	waitgroup := new(sync.WaitGroup)
	waitgroup.Add(2 * b.threadCount)
	for i := 0; i < b.threadCount; i++ {
		go b.processIPs(ipaddrs, waitgroup)
		go b.processURLs(urls, waitgroup, cash)
	}

	ipit := ipcur.Find(bson.M{"local": false}).
		Batch(b.resources.System.BatchSize).
		Prefetch(b.resources.System.Prefetch).
		Iter()

	urlit := urlcur.Find(nil).
		Batch(b.resources.System.BatchSize).
		Prefetch(b.resources.System.Prefetch).
		Iter()

	rwg := new(sync.WaitGroup)
	rwg.Add(2)

	go func(iter *mgo.Iter, ipchan chan string) {
		defer rwg.Done()
		var r datatype_structure.Host
		for iter.Next(&r) {
			if util.RFC1918(r.Ip) {
				continue
			}
			ipchan <- r.Ip
		}
	}(ipit, ipaddrs)

	go func(iter *mgo.Iter, urlchan chan URLShort, ipchan chan string) {
		defer rwg.Done()

		var u URLShort
		for iter.Next(&u) {
			urlchan <- u
		}
	}(urlit, urls, ipaddrs)

	rwg.Wait()
	close(ipaddrs)
	close(urls)

	waitgroup.Wait()
}

// processIPs goes through all of the ips in the ip channel
func (b *Blacklisted) processIPs(ip chan string, waitgroup *sync.WaitGroup) {
	defer waitgroup.Done()
	ipssn := b.resources.DB.Session.Copy()
	defer ipssn.Close()
	cur := ipssn.DB(b.db).C(b.resources.System.BlacklistedConfig.BlacklistTable)

	for {
		ip, ok := <-ip
		if !ok {
			return
		}
		// Append the sources that determined this host
		// was blacklisted
		sourcelist := []string{}

		score := 0
		result := b.ritaBL.CheckHosts([]string{ip}, b.resources.System.BlacklistedConfig.BlacklistDatabase)
		if len(result) > 0 {
			for _, val := range result[0].Results {
				score++
				sourcelist = append(sourcelist, val.HostList)
			}
		}

		if score > 0 {
			err := cur.Insert(&blacklisted.Blacklist{
				BLHash:          fmt.Sprintf("%x", md5.Sum([]byte(ip))),
				BlType:          "ip",
				Score:           score,
				DateChecked:     time.Now().Unix(),
				Host:            ip,
				IsIp:            true,
				IsURL:           false,
				BlacklistSource: sourcelist,
			})

			if err != nil {
				b.log.WithFields(log.Fields{
					"error": err.Error(),
					"cur":   cur,
				}).Error("Error inserting into the blacklist table")
			}
		}
	}
}

// processURLs goes through all of the urls in the url channel
func (b *Blacklisted) processURLs(urls chan URLShort, waitgroup *sync.WaitGroup, cash util.Cache) {
	defer waitgroup.Done()
	urlssn := b.resources.DB.Session.Copy()
	defer urlssn.Close()
	cur := urlssn.DB(b.db).C(b.resources.System.BlacklistedConfig.BlacklistTable)

	for url := range urls {
		actualURL := url.URL
		hsh := fmt.Sprintf("%x", md5.Sum([]byte(actualURL)))
		if cash.Lookup(hsh) {
			continue
		}

		score := 0

		urlList := []string{actualURL}

		if b.safeBrowser != nil {
			result, _ := b.safeBrowser.LookupURLs(urlList)
			if len(result) > 0 && len(result[0]) > 0 {
				score = 1
			}
		}

		if score > 0 {
			err := cur.Insert(&blacklisted.Blacklist{
				BLHash:      hsh,
				BlType:      "url",
				Score:       score,
				DateChecked: time.Now().Unix(),
				Host:        actualURL,
				IsIp:        false,
				IsURL:       true,
			})

			if err != nil {
				b.log.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("Error inserting into the blacklist table")
			}
		}
	}
}
