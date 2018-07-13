package commands

import (
	"encoding/csv"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/activecm/rita/datatypes/blacklist"
	"github.com/activecm/rita/datatypes/structure"
	"github.com/activecm/rita/datatypes/urls"
	"github.com/activecm/rita/resources"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
	"github.com/globalsign/mgo/bson"
)

func init() {
	blURLs := cli.Command{
		Name:      "show-bl-urls",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			humanFlag,
			blConnFlag,
			blSortFlag,
			configFlag,
		},
		Usage:  "Print blacklisted URLs which were visited",
		Action: printBLURLs,
	}

	bootstrapCommands(blURLs)
}

func printBLURLs(c *cli.Context) error {
	db, sort, connected, human, err := parseBLArgs(c)
	if err != nil {
		return err
	}
	res := resources.InitResources(c.String("config"))

	var blURLs []blacklist.BlacklistedURL
	res.DB.Session.DB(db).
		C(res.Config.T.Blacklisted.UrlsTable).
		Find(nil).Sort("-" + sort).All(&blURLs)

	if len(blURLs) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if connected {
		//for each blacklisted url
		for i, blURL := range blURLs {
			//get the ips associated with the url
			var urlEntry urls.URL
			res.DB.Session.DB(db).C(res.Config.T.Urls.UrlsTable).
				Find(bson.M{"url": blURL.Host, "uri": blURL.Resource}).One(&urlEntry)
			ips := urlEntry.IPs
			//and loop over the ips
			for _, ip := range ips {
				//then find all of the hosts which talked to the ip
				var connected []structure.UniqueConnection
				res.DB.Session.DB(db).
					C(res.Config.T.Structure.UniqueConnTable).Find(
					bson.M{"dst": ip},
				).All(&connected)
				//and aggregate the source ip addresses
				for _, uconn := range connected {
					blURLs[i].ConnectedHosts = append(blURLs[i].ConnectedHosts, uconn.Src)
				}
			}
		}
	}
	if human {
		err = showBLURLsHuman(blURLs, connected)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	} else {
		err = showBLURLs(blURLs, connected)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	}

	return nil
}

func showBLURLs(urls []blacklist.BlacklistedURL, connectedHosts bool) error {
	csvWriter := csv.NewWriter(os.Stdout)
	headers := []string{"Host", "Resource", "Connections", "Unique Connections", "Total Bytes", "Lists"}
	if connectedHosts {
		headers = append(headers, "Sources")
	}
	csvWriter.Write(headers)
	for _, url := range urls {
		sort.Strings(url.Lists)
		serialized := []string{
			url.Host,
			url.Resource,
			strconv.Itoa(url.Connections),
			strconv.Itoa(url.UniqueConnections),
			strconv.Itoa(url.TotalBytes),
			strings.Join(url.Lists, " "),
		}
		if connectedHosts {
			sort.Strings(url.ConnectedHosts)
			serialized = append(serialized, strings.Join(url.ConnectedHosts, " "))
		}
		csvWriter.Write(serialized)
	}
	csvWriter.Flush()
	return nil
}

func showBLURLsHuman(urls []blacklist.BlacklistedURL, connectedHosts bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	headers := []string{"Host", "Resource", "Connections", "Unique Connections", "Total Bytes", "Lists"}
	if connectedHosts {
		headers = append(headers, "Sources")
	}
	table.SetHeader(headers)
	for _, url := range urls {
		sort.Strings(url.Lists)
		serialized := []string{
			url.Host,
			url.Resource,
			strconv.Itoa(url.Connections),
			strconv.Itoa(url.UniqueConnections),
			strconv.Itoa(url.TotalBytes),
			strings.Join(url.Lists, " "),
		}
		if connectedHosts {
			sort.Strings(url.ConnectedHosts)
			serialized = append(serialized, strings.Join(url.ConnectedHosts, " "))
		}
		table.Append(serialized)
	}
	table.Render()
	return nil
}
