package commands

import (
	"encoding/csv"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

const endl = "\r\n"

func init() {
	blSourceIPs := cli.Command{
		Name:      "show-bl-source-ips",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			humanFlag,
			blConnFlag,
			blSortFlag,
			configFlag,
		},
		Usage:  "Print blacklisted IPs which initiated connections",
		Action: printBLSourceIPs,
	}

	blDestIPs := cli.Command{
		Name:      "show-bl-dest-ips",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			humanFlag,
			blConnFlag,
			blSortFlag,
			configFlag,
		},
		Usage:  "Print blacklisted IPs which received connections",
		Action: printBLDestIPs,
	}

	bootstrapCommands(blSourceIPs, blDestIPs)
}

func parseBLArgs(c *cli.Context) (string, string, bool, bool, error) {
	db := c.Args().Get(0)
	sort := c.String("sort")
	connected := c.Bool("connected")
	human := c.Bool("human-readable")
	if db == "" {
		return db, sort, connected, human, cli.NewExitError("Specify a database", -1)
	}
	if sort != "conn" && sort != "uconn" && sort != "total_bytes" {
		return db, sort, connected, human, cli.NewExitError("Invalid option passed to sort flag", -1)
	}
	return db, sort, connected, human, nil
}

func printBLSourceIPs(c *cli.Context) error {
	db, sort, connected, human, err := parseBLArgs(c)
	if err != nil {
		return err
	}
	res := resources.InitResources(c.String("config"))

	var blIPs []host.AnalysisView

	blacklistFindQuery := bson.M{
		"$and": []bson.M{
			bson.M{"blacklisted": true},
			bson.M{"count_src": bson.M{"$gt": 0}},
		}}

	res.DB.Session.DB(db).
		C(res.Config.T.Structure.HostTable).
		Find(blacklistFindQuery).Sort("-" + sort).All(&blIPs)

	if len(blIPs) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if connected {
		for i, entry := range blIPs {
			var connected []uconn.AnalysisView
			res.DB.Session.DB(db).
				C(res.Config.T.Structure.UniqueConnTable).Find(
				bson.M{"src": entry.Host},
			).All(&connected)
			for _, uconn := range connected {
				blIPs[i].ConnectedHosts = append(blIPs[i].ConnectedHosts, uconn.Dst)
			}
		}
	}

	if human {
		err = showBLIPsHuman(blIPs, connected, true)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	} else {
		err = showBLIPs(blIPs, connected, true)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	}
	return nil
}

func printBLDestIPs(c *cli.Context) error {
	db, sort, connected, human, err := parseBLArgs(c)
	if err != nil {
		return err
	}
	res := resources.InitResources(c.String("config"))

	var blIPs []host.AnalysisView

	blacklistFindQuery := bson.M{
		"$and": []bson.M{
			bson.M{"blacklisted": true},
			bson.M{"count_dst": bson.M{"$gt": 0}},
		}}

	res.DB.Session.DB(db).
		C(res.Config.T.Structure.HostTable).
		Find(blacklistFindQuery).Sort("-" + sort).All(&blIPs)

	if len(blIPs) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if connected {
		for i, entry := range blIPs {
			var connected []uconn.AnalysisView
			res.DB.Session.DB(db).
				C(res.Config.T.Structure.UniqueConnTable).Find(
				bson.M{"dst": entry.Host},
			).All(&connected)
			for _, uconn := range connected {
				blIPs[i].ConnectedHosts = append(blIPs[i].ConnectedHosts, uconn.Src)
			}
		}
	}

	if human {
		err = showBLIPsHuman(blIPs, connected, false)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	} else {
		err = showBLIPs(blIPs, connected, false)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	}
	return nil
}

func showBLIPs(ips []host.AnalysisView, connectedHosts, source bool) error {
	csvWriter := csv.NewWriter(os.Stdout)
	headers := []string{"IP", "Connections", "Unique Connections", "Total Bytes"}
	if connectedHosts {
		if source {
			headers = append(headers, "Destinations")
		} else {
			headers = append(headers, "Sources")
		}
	}
	csvWriter.Write(headers)
	for _, entry := range ips {

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

func showBLIPsHuman(ips []host.AnalysisView, connectedHosts, source bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	headers := []string{"IP", "Connections", "Unique Connections", "Total Bytes"}
	if connectedHosts {
		if source {
			headers = append(headers, "Destinations")
		} else {
			headers = append(headers, "Sources")
		}
	}
	table.SetHeader(headers)
	for _, entry := range ips {

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
