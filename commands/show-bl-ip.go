package commands

import (
	"encoding/csv"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/blacklist"
	"github.com/ocmdev/rita/datatypes/structure"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
	"gopkg.in/mgo.v2/bson"
)

const blacklistListsTemplate = "{{range $idx, $list := .Lists}}{{if $idx}} {{end}}{{ $list }}{{end}}"
const endl = "\r\n"

func init() {
	blSourceIPs := cli.Command{
		Name: "show-bl-source-ips",
		Flags: []cli.Flag{
			databaseFlag,
			humanFlag,
			blConnFlag,
			blSortFlag,
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
			blConnFlag,
			blSortFlag,
			configFlag,
		},
		Usage:  "Print blacklisted IPs which recieved connections",
		Action: printBLDestIPs,
	}

	bootstrapCommands(blSourceIPs, blDestIPs)
}

func parseBLArgs(c *cli.Context) (string, string, bool, bool, error) {
	db := c.String("database")
	sort := c.String("sort")
	connected := c.Bool("connected")
	human := c.Bool("human-readable")
	if db == "" {
		return db, sort, connected, human, cli.NewExitError("Specify a database with -d", -1)
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

func showBLIPs(ips []blacklist.BlacklistedIP, connectedHosts, source bool) error {
	csvWriter := csv.NewWriter(os.Stdout)
	headers := []string{"IP", "Connections", "Unique Connections", "Total Bytes", "Lists"}
	if connectedHosts {
		if source {
			headers = append(headers, "Destinations")
		} else {
			headers = append(headers, "Sources")
		}
	}
	csvWriter.Write(headers)
	for _, ip := range ips {
		sort.Strings(ip.Lists)
		serialized := []string{
			ip.IP,
			strconv.Itoa(ip.Connections),
			strconv.Itoa(ip.UniqueConnections),
			strconv.Itoa(ip.TotalBytes),
			strings.Join(ip.Lists, " "),
		}
		if connectedHosts {
			sort.Strings(ip.ConnectedHosts)
			serialized = append(serialized, strings.Join(ip.ConnectedHosts, " "))
		}
		csvWriter.Write(serialized)
	}
	csvWriter.Flush()
	return nil
}

func showBLIPsHuman(ips []blacklist.BlacklistedIP, connectedHosts, source bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	headers := []string{"IP", "Connections", "Unique Connections", "Total Bytes", "Lists"}
	if connectedHosts {
		if source {
			headers = append(headers, "Destinations")
		} else {
			headers = append(headers, "Sources")
		}
	}
	table.SetHeader(headers)
	for _, ip := range ips {
		sort.Strings(ip.Lists)
		serialized := []string{
			ip.IP,
			strconv.Itoa(ip.Connections),
			strconv.Itoa(ip.UniqueConnections),
			strconv.Itoa(ip.TotalBytes),
			strings.Join(ip.Lists, " "),
		}
		if connectedHosts {
			sort.Strings(ip.ConnectedHosts)
			serialized = append(serialized, strings.Join(ip.ConnectedHosts, " "))
		}
		table.Append(serialized)
	}
	table.Render()
	return nil
}
