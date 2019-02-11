package commands

import (
	"encoding/csv"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {

	blHostnames := cli.Command{
		Name:      "show-bl-hostnames",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			humanFlag,
			blConnFlag,
			blSortFlag,
			configFlag,
		},
		Usage:  "Print blacklisted hostnames which received connections",
		Action: printBLHostnames,
	}

	bootstrapCommands(blHostnames)
}

func printBLHostnames(c *cli.Context) error {
	db, sort, connected, human, err := parseBLArgs(c)
	if err != nil {
		return err
	}
	res := resources.InitResources(c.String("config"))
	res.DB.SelectDB(db) //so we can use the dns.GetIPsFromHost method

	var blHosts []hostname.AnalysisView
	res.DB.Session.DB(db).
		C(res.Config.T.DNS.HostnamesTable).
		Find(bson.M{"blacklisted": true}).Sort("-" + sort).All(&blHosts)

	if len(blHosts) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if connected {
		//for each blacklisted host
		for i, entry := range blHosts {

			//and loop over the ips associated with the host
			for _, ip := range entry.IPs {
				//then find all of the hosts which talked to the ip
				var connected []uconn.AnalysisView
				res.DB.Session.DB(db).
					C(res.Config.T.Structure.UniqueConnTable).Find(
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

func showBLHostnames(hostnames []hostname.AnalysisView, connectedHosts bool) error {
	csvWriter := csv.NewWriter(os.Stdout)
	headers := []string{"Hostname", "Connections", "Unique Connections", "Total Bytes"}
	if connectedHosts {
		headers = append(headers, "Sources")
	}
	csvWriter.Write(headers)
	for _, entry := range hostnames {

		serialized := []string{
			entry.Host,
			strconv.Itoa(entry.Connections),
			strconv.Itoa(entry.UniqueConnections),
			strconv.Itoa(entry.TotalBytes),
		}
		if connectedHosts {
			sort.Strings(entry.ConnectedHosts)
			serialized = append(serialized, strings.Join(entry.ConnectedHosts, " "))
		}
		csvWriter.Write(serialized)
	}
	csvWriter.Flush()

	return nil
}

func showBLHostnamesHuman(hostnames []hostname.AnalysisView, connectedHosts bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	headers := []string{"Hostname", "Connections", "Unique Connections", "Total Bytes"}
	if connectedHosts {
		headers = append(headers, "Sources")
	}
	table.SetHeader(headers)
	for _, entry := range hostnames {

		serialized := []string{
			entry.Host,
			strconv.Itoa(entry.Connections),
			strconv.Itoa(entry.UniqueConnections),
			strconv.Itoa(entry.TotalBytes),
		}
		if connectedHosts {
			sort.Strings(entry.ConnectedHosts)
			serialized = append(serialized, strings.Join(entry.ConnectedHosts, " "))
		}
		table.Append(serialized)
	}
	table.Render()
	return nil
}
