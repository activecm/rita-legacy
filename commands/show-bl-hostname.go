package commands

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/activecm/rita/pkg/hostname"
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
			configFlag,
		},
		Usage:  "Print blacklisted hostnames which received connections",
		Action: printBLHostnames,
	}

	bootstrapCommands(blHostnames)
}

func printBLHostnames(c *cli.Context) error {
	db := c.Args().Get(0)

	if db == "" {
		return cli.NewExitError("Specify a database", -1)
	}

	res := resources.InitResources(c.String("config"))
	res.DB.SelectDB(db)

	data := getBlacklistedHostnameResultsView(res, "conn_count", 1000)

	if len(data) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if c.Bool("human-readable") {
		err := showBLHostnamesHuman(data)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	} else {
		err := showBLHostnames(data)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	}

	return nil
}

func showBLHostnames(hostnames []hostname.AnalysisView) error {
	csvWriter := csv.NewWriter(os.Stdout)
	headers := []string{"Host", "Connections", "Unique Connections", "Total Bytes", "Sources"}

	csvWriter.Write(headers)
	for _, entry := range hostnames {

		serialized := []string{
			entry.Host,
			strconv.Itoa(entry.Connections),
			strconv.Itoa(entry.UniqueConnections),
			strconv.Itoa(entry.TotalBytes),
		}

		sort.Strings(entry.ConnectedHosts)
		serialized = append(serialized, strings.Join(entry.ConnectedHosts, " "))

		csvWriter.Write(serialized)
	}
	csvWriter.Flush()

	return nil
}

func showBLHostnamesHuman(hostnames []hostname.AnalysisView) error {
	table := tablewriter.NewWriter(os.Stdout)
	headers := []string{"Hostname", "Connections", "Unique Connections", "Total Bytes", "Sources"}

	table.SetHeader(headers)
	for _, entry := range hostnames {

		serialized := []string{
			entry.Host,
			strconv.Itoa(entry.Connections),
			strconv.Itoa(entry.UniqueConnections),
			strconv.Itoa(entry.TotalBytes),
		}

		sort.Strings(entry.ConnectedHosts)
		serialized = append(serialized, strings.Join(entry.ConnectedHosts, " "))

		table.Append(serialized)
	}
	table.Render()
	return nil
}

//getBlacklistedHostnameResultsView ....
func getBlacklistedHostnameResultsView(res *resources.Resources, sort string, limit int) []hostname.AnalysisView {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	blHostsQuery := []bson.M{
		bson.M{"$match": bson.M{"blacklisted": true}},
		bson.M{"$unwind": "$dat"},
		bson.M{"$project": bson.M{"host": 1, "ip": "$dat.ips"}},
		bson.M{"$unwind": "$ip"},
		bson.M{"$lookup": bson.M{
			"from":         "uconn",
			"localField":   "ip",
			"foreignField": "dst",
			"as":           "uconn",
		}},
		bson.M{"$unwind": "$uconn"},
		bson.M{"$unwind": "$uconn.dat"},
		bson.M{"$project": bson.M{"host": 1, "conns": "$uconn.dat.count", "bytes": "$uconn.dat.tbytes", "ip": "$uconn.src"}},
		bson.M{"$group": bson.M{
			"_id":         "$host",
			"ips":         bson.M{"$addToSet": "$ip"},
			"conn_count":  bson.M{"$sum": "$conns"},
			"total_bytes": bson.M{"$sum": "$bytes"},
		}},
		bson.M{"$sort": bson.M{sort: -1}},
		bson.M{"$limit": limit},
		bson.M{"$project": bson.M{
			"_id":         0,
			"host":        "$_id",
			"uconn_count": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$ips", []interface{}{}}}},
			"ips":         1,
			"conn_count":  1,
			"total_bytes": 1,
		}},
	}

	var blHosts []hostname.AnalysisView

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.HostnamesTable).Pipe(blHostsQuery).All(&blHosts)

	if err != nil {
		fmt.Println(err)
	}

	return blHosts

}
