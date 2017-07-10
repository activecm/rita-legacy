package commands

import (
	"encoding/csv"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ocmdev/rita/analysis/dns"
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/blacklist"
	"github.com/ocmdev/rita/datatypes/structure"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
	"gopkg.in/mgo.v2/bson"
)

func init() {

	blHostnames := cli.Command{
		Name: "show-bl-hostnames",
		Flags: []cli.Flag{
			databaseFlag,
			humanFlag,
			blConnFlag,
			blSortFlag,
			configFlag,
		},
		Usage:  "Print blacklisted hostnames which recieved connections",
		Action: printBLHostnames,
	}

	bootstrapCommands(blHostnames)
}

func printBLHostnames(c *cli.Context) error {
	db, sort, connected, human, err := parseBLArgs(c)
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

	if human {
		err = showBLHostnamesHuman(blHosts, connected)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	} else {
		err = showBLHostnames(blHosts, connected)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	}

	return nil
}

func showBLHostnames(hostnames []blacklist.BlacklistedHostname, connectedHosts bool) error {
	csvWriter := csv.NewWriter(os.Stdout)
	headers := []string{"Hostname", "Connections", "Unique Connections", "Total Bytes", "Lists"}
	if connectedHosts {
		headers = append(headers, "Sources")
	}
	csvWriter.Write(headers)
	for _, hostname := range hostnames {
		sort.Strings(hostname.Lists)
		serialized := []string{
			hostname.Hostname,
			strconv.Itoa(hostname.Connections),
			strconv.Itoa(hostname.UniqueConnections),
			strconv.Itoa(hostname.TotalBytes),
			strings.Join(hostname.Lists, " "),
		}
		if connectedHosts {
			sort.Strings(hostname.ConnectedHosts)
			serialized = append(serialized, strings.Join(hostname.ConnectedHosts, " "))
		}
		csvWriter.Write(serialized)
	}
	csvWriter.Flush()

	return nil
}

func showBLHostnamesHuman(hostnames []blacklist.BlacklistedHostname, connectedHosts bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	headers := []string{"Hostname", "Connections", "Unique Connections", "Total Bytes", "Lists"}
	if connectedHosts {
		headers = append(headers, "Sources")
	}
	table.SetHeader(headers)
	for _, hostname := range hostnames {
		sort.Strings(hostname.Lists)
		serialized := []string{
			hostname.Hostname,
			strconv.Itoa(hostname.Connections),
			strconv.Itoa(hostname.UniqueConnections),
			strconv.Itoa(hostname.TotalBytes),
			strings.Join(hostname.Lists, " "),
		}
		if connectedHosts {
			sort.Strings(hostname.ConnectedHosts)
			serialized = append(serialized, strings.Join(hostname.ConnectedHosts, " "))
		}
		table.Append(serialized)
	}
	table.Render()
	return nil
}
