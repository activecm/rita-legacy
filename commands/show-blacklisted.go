package commands

import (
	"fmt"

	"github.com/ocmdev/rita/analysis/dns"
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/blacklist"
	"github.com/ocmdev/rita/datatypes/structure"
	"github.com/ocmdev/rita/datatypes/urls"
	"github.com/urfave/cli"
	"gopkg.in/mgo.v2/bson"
)

func init() {
	sortFlag := cli.StringFlag{
		Name:  "sort, s",
		Usage: "Sort by conn (# of connections), uconn (# of unique connections), total_bytes (# of bytes)",
		Value: "conn",
	}
	connFlag := cli.BoolFlag{
		Name:  "connected, C",
		Usage: "Show hosts which were connected to this blacklisted entry",
	}

	blSourceIPs := cli.Command{
		Name: "show-bl-source-ips",
		Flags: []cli.Flag{
			databaseFlag,
			humanFlag,
			connFlag,
			sortFlag,
			configFlag,
		},
		Usage:  "Print blacklisted IPs which initiated connections",
		Action: printBLSourceIPs,
	}

	blDestIPs := cli.Command{
		Name: "show-bl-dest-ips",
		Flags: []cli.Flag{
			databaseFlag,
			humanFlag,
			connFlag,
			sortFlag,
			configFlag,
		},
		Usage:  "Print blacklisted IPs which recieved connections",
		Action: printBLDestIPs,
	}

	blHostnames := cli.Command{
		Name: "show-bl-hostnames",
		Flags: []cli.Flag{
			databaseFlag,
			humanFlag,
			connFlag,
			sortFlag,
			configFlag,
		},
		Usage:  "Print blacklisted hostnames which recieved connections",
		Action: printBLHostnames,
	}

	blURLs := cli.Command{
		Name: "show-bl-urls",
		Flags: []cli.Flag{
			databaseFlag,
			humanFlag,
			connFlag,
			sortFlag,
			configFlag,
		},
		Usage:  "Print blacklisted URLs which were visited",
		Action: printBLURLs,
	}

	bootstrapCommands(blSourceIPs, blDestIPs, blHostnames, blURLs)
}

func parseArgs(c *cli.Context) (string, string, bool, error) {
	db := c.String("database")
	sort := c.String("sort")
	connected := c.Bool("connected")
	if db == "" {
		return db, sort, connected, cli.NewExitError("Specify a database with -d", -1)
	}
	if sort != "conn" && sort != "uconn" && sort != "total_bytes" {
		return db, sort, connected, cli.NewExitError("Invalid option passed to sort flag", -1)
	}
	return db, sort, connected, nil
}

func printBLSourceIPs(c *cli.Context) error {
	db, sort, connected, err := parseArgs(c)
	if err != nil {
		return err
	}
	res := database.InitResources(c.String("config"))

	var blIPs []blacklist.BlacklistedIP
	res.DB.Session.DB(db).
		C(res.System.BlacklistedConfig.SourceIPsTable).
		Find(nil).Sort("-" + sort).All(&blIPs)

	if len(blIPs) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if connected {
		for i, ip := range blIPs {
			var connected []structure.UniqueConnection
			res.DB.Session.DB(db).
				C(res.System.StructureConfig.UniqueConnTable).Find(
				bson.M{"src": ip.IP},
			).All(&connected)
			for _, uconn := range connected {
				blIPs[i].ConnectedHosts = append(blIPs[i].ConnectedHosts, uconn.Dst)
			}
		}
	}

	for _, entry := range blIPs {
		fmt.Println(entry)
	}
	return nil
}

func printBLDestIPs(c *cli.Context) error {
	db, sort, connected, err := parseArgs(c)
	if err != nil {
		return err
	}
	res := database.InitResources(c.String("config"))

	var blIPs []blacklist.BlacklistedIP
	res.DB.Session.DB(db).
		C(res.System.BlacklistedConfig.DestIPsTable).
		Find(nil).Sort("-" + sort).All(&blIPs)

	if len(blIPs) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if connected {
		for i, ip := range blIPs {
			var connected []structure.UniqueConnection
			res.DB.Session.DB(db).
				C(res.System.StructureConfig.UniqueConnTable).Find(
				bson.M{"dst": ip.IP},
			).All(&connected)
			for _, uconn := range connected {
				blIPs[i].ConnectedHosts = append(blIPs[i].ConnectedHosts, uconn.Src)
			}
		}
	}

	for _, entry := range blIPs {
		fmt.Println(entry)
	}
	return nil
}

func printBLHostnames(c *cli.Context) error {
	db, sort, connected, err := parseArgs(c)
	if err != nil {
		return err
	}
	res := database.InitResources(c.String("config"))
	res.DB.SelectDB(db) //so we can use the dns.GetIPsFromHost method

	var blHosts []blacklist.BlacklistedHostname
	res.DB.Session.DB(db).
		C(res.System.BlacklistedConfig.HostnamesTable).
		Find(nil).Sort("-" + sort).All(&blHosts)

	if len(blHosts) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if connected {
		//for each blacklisted host
		for i, host := range blHosts {
			//get the ips associated with the host
			ips := dns.GetIPsFromHost(res, host.Hostname)
			//and loop over the ips
			for _, ip := range ips {
				//then find all of the hosts which talked to the ip
				var connected []structure.UniqueConnection
				res.DB.Session.DB(db).
					C(res.System.StructureConfig.UniqueConnTable).Find(
					bson.M{"dst": ip},
				).All(&connected)
				//and aggregate the source ip addresses
				for _, uconn := range connected {
					blHosts[i].ConnectedHosts = append(blHosts[i].ConnectedHosts, uconn.Src)
				}
			}
		}
	}

	for _, entry := range blHosts {
		fmt.Println(entry)
	}
	return nil
}

func printBLURLs(c *cli.Context) error {
	db, sort, connected, err := parseArgs(c)
	if err != nil {
		return err
	}
	res := database.InitResources(c.String("config"))

	var blURLs []blacklist.BlacklistedURL
	res.DB.Session.DB(db).
		C(res.System.BlacklistedConfig.UrlsTable).
		Find(nil).Sort("-" + sort).All(&blURLs)

	if len(blURLs) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if connected {
		//for each blacklisted url
		for i, blURL := range blURLs {
			//get the ips associated with the url
			var urlEntry urls.URL
			res.DB.Session.DB(db).C(res.System.UrlsConfig.UrlsTable).
				Find(bson.M{"url": blURL.Host, "uri": blURL.Resource}).One(&urlEntry)
			ips := urlEntry.IPs
			//and loop over the ips
			for _, ip := range ips {
				//then find all of the hosts which talked to the ip
				var connected []structure.UniqueConnection
				res.DB.Session.DB(db).
					C(res.System.StructureConfig.UniqueConnTable).Find(
					bson.M{"dst": ip},
				).All(&connected)
				//and aggregate the source ip addresses
				for _, uconn := range connected {
					blURLs[i].ConnectedHosts = append(blURLs[i].ConnectedHosts, uconn.Src)
				}
			}
		}
	}

	for _, entry := range blURLs {
		fmt.Println(entry)
	}
	return nil
}
