package blacklisted

import (
	"crypto/md5"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ocmdev/rita/database"

	"github.com/google/safebrowsing"

	"github.com/ocmdev/rita/util"

	"github.com/ocmdev/rita-blacklist"
	"github.com/ocmdev/rita/datatypes/blacklisted"
	datatype_structure "github.com/ocmdev/rita/datatypes/structure"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type (
	// Blacklisted provides a handle for the blacklist module
	Blacklisted struct {
		db              string                    // database name (customer)
		batch_size      int                       // BatchSize
		prefetch        float64                   // Prefetch
		resources       *database.Resources       // resources
		log             *log.Logger               // logger
		channel_size    int                       // channel size
		thread_count    int                       // Thread count
		blacklist_table string                    // Name of blacklist table
		safeBrowser     *safebrowsing.SafeBrowser // Google safebrowsing api
		ritaBL          *blacklist.BlackList      // Blacklisted host database
	}

	// UrlShort is a shortened version of the URL datatype that only accounts
	// for the IP and url (hostname)
	UrlShort struct {
		Url string   `bson:"host"`
		IPs []string `bson:"ips"`
	}
)

func BuildBlacklistedCollection(res *database.Resources) {
	collection_name := res.System.BlacklistedConfig.BlacklistTable
	collection_keys := []string{"bl_hash", "host"}
	error_check := res.DB.CreateCollection(collection_name, collection_keys)
	if error_check != "" {
		res.Log.Error("Failed: ", collection_name, error_check)
		return
	}
	newBlacklisted(res).run()
}

// New will create a new blacklisted module
func newBlacklisted(res *database.Resources) *Blacklisted {

	// Initialize the hook to google's safebrowsing api.
	sbConfig := safebrowsing.Config{
		APIKey: res.System.SafeBrowsing.APIKey,
		DBPath: res.System.SafeBrowsing.Database,
		Logger: res.Log.Writer(),
	}
	sb, err := safebrowsing.NewSafeBrowser(sbConfig)
	if err != nil {
		res.Log.WithField("Error", err).Error("Error opening safe browser API")
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
		} else {
			res.Log.WithField("Error", err).Error("Error opening rita-blacklist hook")
		}
	}

	return &Blacklisted{
		db:              res.DB.GetSelectedDB(),
		batch_size:      res.System.BatchSize,
		prefetch:        res.System.Prefetch,
		resources:       res,
		log:             res.Log,
		channel_size:    res.System.BlacklistedConfig.ChannelSize,
		thread_count:    res.System.BlacklistedConfig.ThreadCount,
		blacklist_table: res.System.BlacklistedConfig.BlacklistTable,
		safeBrowser:     sb,
		ritaBL:          ritabl,
	}
}

// Run runs the module
func (b *Blacklisted) run() {
	start := time.Now()
	ipssn := b.resources.DB.Session.Copy()
	defer ipssn.Close()
	urlssn := b.resources.DB.Session.Copy()
	defer urlssn.Close()

	// build up cursors
	ipcur := ipssn.DB(b.db).C(b.resources.System.StructureConfig.HostTable)
	urlcur := urlssn.DB(b.db).C(b.resources.System.UrlsConfig.HostnamesTable)

	runtime.GOMAXPROCS(runtime.NumCPU())
	ipaddrs := make(chan string, b.channel_size)
	urls := make(chan UrlShort, b.channel_size)
	cash := util.NewCache()
	waitgroup := new(sync.WaitGroup)
	waitgroup.Add(2 * b.thread_count)
	for i := 0; i < b.thread_count; i++ {
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

	go func(iter *mgo.Iter, urlchan chan UrlShort, ipchan chan string) {
		defer rwg.Done()

		var u UrlShort
		for iter.Next(&u) {
			for _, ip := range u.IPs {
				if util.RFC1918(ip) {
					continue
				}
				ipchan <- ip
			}
			urlchan <- u
		}
	}(urlit, urls, ipaddrs)

	rwg.Wait()
	close(ipaddrs)
	close(urls)

	b.log.Info("Lookups complete waiting on checks to run")
	waitgroup.Wait()
	b.log.WithFields(log.Fields{
		"time_elapsed": time.Since(start),
	}).Info("Blacklist analysis completed")
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

		score := 0
		result := b.ritaBL.CheckHosts([]string{ip}, b.resources.System.BlacklistedConfig.BlacklistDatabase)
		if len(result) > 0 {
			score = len(result[0].Results)
		}

		if score > 0 {
			err := cur.Insert(&blacklisted.Blacklist{
				BLHash:      fmt.Sprintf("%x", md5.Sum([]byte(ip))),
				BlType:      "ip",
				Score:       score,
				DateChecked: time.Now().Unix(),
				Host:        ip,
				IsIp:        true,
				IsUrl:       false,
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
func (b *Blacklisted) processURLs(urls chan UrlShort, waitgroup *sync.WaitGroup, cash util.Cache) {
	defer waitgroup.Done()
	urlssn := b.resources.DB.Session.Copy()
	defer urlssn.Close()
	cur := urlssn.DB(b.db).C(b.resources.System.BlacklistedConfig.BlacklistTable)

	for url := range urls {
		actualURL := url.Url
		hsh := fmt.Sprintf("%x", md5.Sum([]byte(actualURL)))
		if cash.Lookup(hsh) {
			continue
		}

		score := 0

		urlList := []string{actualURL}
		result, _ := b.safeBrowser.LookupURLs(urlList)
		if len(result) > 0 && len(result[0]) > 0 {
			for _ = range url.IPs {
				score += 1
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
				IsUrl:       true,
			})

			if err != nil {
				b.log.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("Error inserting into the blacklist table")
			}
		}
	}
}
