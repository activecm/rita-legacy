package commands

import (
	"encoding/csv"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	blSourceIPs := cli.Command{
		Name:      "show-bl-source-ips",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			humanFlag,
			blConnFlag,
			blSortFlag,
			configFlag,
			limitFlag,
			noLimitFlag,
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
	if sort != "conn_count" && sort != "total_bytes" {
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
	res.DB.SelectDB(db)

	match := bson.M{
		"$and": []bson.M{
			bson.M{"blacklisted": true},
			bson.M{"dat.count_src": bson.M{"$gt": 0}},
		}}
	limit := c.Int("limit")
	data, err := getBlacklistedIPsResultsView(res, sort, c.Bool("no-limit"), limit, match, "src", "dst")

	if err != nil {
		res.Log.Error(err)
		return cli.NewExitError(err, -1)
	}

	if len(data) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if human {
		err = showBLIPsHuman(data, connected, true)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	} else {
		err = showBLIPs(data, connected, true)
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
	res.DB.SelectDB(db)

	match := bson.M{
		"$and": []bson.M{
			bson.M{"blacklisted": true},
			bson.M{"dat.count_dst": bson.M{"$gt": 0}},
		}}

	limit := c.Int("limit")
	data, err := getBlacklistedIPsResultsView(res, sort, c.Bool("no-limit"), limit, match, "dst", "src")

	if err != nil {
		res.Log.Error(err)
		return cli.NewExitError(err, -1)
	}

	if len(data) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if human {
		err = showBLIPsHuman(data, connected, false)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	} else {
		err = showBLIPs(data, connected, false)
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

//getBlaclistedIPsResultsView
func getBlacklistedIPsResultsView(res *resources.Resources, sort string, noLimit bool, limit int, match bson.M, field1 string, field2 string) ([]host.AnalysisView, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var blIPs []host.AnalysisView

	blIPQuery := []bson.M{
		bson.M{"$match": match},
		bson.M{"$project": bson.M{"host": "$ip"}},
		bson.M{"$lookup": bson.M{
			"from":         "uconn",
			"localField":   "host",
			"foreignField": field1,
			"as":           "u",
		}},
		bson.M{"$unwind": "$u"},
		bson.M{"$unwind": "$u.dat"},
		bson.M{"$project": bson.M{"host": 1, "conns": "$u.dat.count", "bytes": "$u.dat.tbytes", "ip": ("$u." + field2)}},
		bson.M{"$group": bson.M{
			"_id":         "$host",
			"host":        bson.M{"$first": "$host"},
			"ips":         bson.M{"$addToSet": "$ip"},
			"conn_count":  bson.M{"$sum": "$conns"},
			"total_bytes": bson.M{"$sum": "$bytes"},
		}},
		bson.M{"$sort": bson.M{sort: -1}},
		bson.M{"$project": bson.M{
			"_id":         0,
			"uconn_count": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$ips", []interface{}{}}}},
			"ips":         1,
			"conn_count":  1,
			"host":        1,
			"total_bytes": 1,
		}},
	}

	if !noLimit {
		blIPQuery = append(blIPQuery, bson.M{"$limit": limit})
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.HostTable).Pipe(blIPQuery).AllowDiskUse().All(&blIPs)

	return blIPs, err

}
