package commands

import (
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
			limitFlag,
			noLimitFlag,
			delimFlag,
			netNamesFlag,
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

	data, err := getBlacklistedHostnameResultsView(res, "conn_count", c.Int("limit"), c.Bool("no-limit"))

	if err != nil {
		res.Log.Error(err)
		return cli.NewExitError(err, -1)
	}

	if len(data) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if c.Bool("human-readable") {
		err = showBLHostnamesHuman(data, c.Bool("network-names"))
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	} else {
		err = showBLHostnames(data, c.String("delimiter"), c.Bool("network-names"))
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	}

	return nil
}

func showBLHostnames(hostnames []hostname.AnalysisView, delim string, netNames bool) error {
	headers := []string{"Host", "Connections", "Unique Connections", "Total Bytes", "Sources"}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headers, delim))
	for _, entry := range hostnames {

		serialized := []string{
			entry.Host,
			strconv.Itoa(entry.Connections),
			strconv.Itoa(entry.UniqueConnections),
			strconv.Itoa(entry.TotalBytes),
		}

		var sourceIPs []string
		if netNames {
			for _, connectedUniqIP := range entry.ConnectedHosts {
				escapedNetName := strings.ReplaceAll(connectedUniqIP.NetworkName, " ", "_")
				escapedNetName = strings.ReplaceAll(escapedNetName, ":", "_")
				connectedIPStr := escapedNetName + ":" + connectedUniqIP.IP
				sourceIPs = append(sourceIPs, connectedIPStr)
			}
		} else {
			for _, connectedUniqIP := range entry.ConnectedHosts {
				sourceIPs = append(sourceIPs, connectedUniqIP.IP)
			}
		}

		sort.Strings(sourceIPs)
		serialized = append(serialized, strings.Join(sourceIPs, " "))

		fmt.Println(
			strings.Join(
				serialized,
				delim,
			),
		)
	}

	return nil
}

func showBLHostnamesHuman(hostnames []hostname.AnalysisView, netNames bool) error {
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

		var sourceIPs []string
		if netNames {
			for _, connectedUniqIP := range entry.ConnectedHosts {
				escapedNetName := strings.ReplaceAll(connectedUniqIP.NetworkName, " ", "_")
				escapedNetName = strings.ReplaceAll(escapedNetName, ":", "_")
				connectedIPStr := escapedNetName + ":" + connectedUniqIP.IP
				sourceIPs = append(sourceIPs, connectedIPStr)
			}
		} else {
			for _, connectedUniqIP := range entry.ConnectedHosts {
				sourceIPs = append(sourceIPs, connectedUniqIP.IP)
			}
		}

		sort.Strings(sourceIPs)
		serialized = append(serialized, strings.Join(sourceIPs, " "))

		table.Append(serialized)
	}
	table.Render()
	return nil
}

//getBlacklistedHostnameResultsView ....
func getBlacklistedHostnameResultsView(res *resources.Resources, sort string, limit int, noLimit bool) ([]hostname.AnalysisView, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	blHostsQuery := []bson.M{
		// find blacklisted hostnames and the IPs associated with them
		bson.M{"$match": bson.M{"blacklisted": true}},
		bson.M{"$project": bson.M{
			"host":    1,
			"dat.ips": 1,
		}},
		// aggregate over time/ chunks
		bson.M{"$unwind": "$dat"},
		// remove duplicate ips associated with each hostname
		bson.M{"$unwind": "$dat.ips"},
		// remove network_name as it may not be consistent with
		// network_uuid and we don't need to display it
		bson.M{"$project": bson.M{"dat.ips.network_name": 0}},
		bson.M{"$group": bson.M{
			"_id": "$host",
			"ips": bson.M{"$addToSet": "$dat.ips"},
		}},
		bson.M{"$unwind": "$ips"},
		// find out which IPs connected to each hostname via uconn
		bson.M{"$lookup": bson.M{
			"from": "uconn",
			"let":  bson.M{"ip": "$ips.ip", "network_uuid": "$ips.network_uuid"},
			"pipeline": []bson.M{{"$match": bson.M{"$expr": bson.M{
				"$and": []bson.M{
					{"$eq": []string{"$dst", "$$ip"}},
					{"$eq": []string{"$dst_network_uuid", "$$network_uuid"}},
				},
			}}}},
			"as": "uconn",
		}},
		bson.M{"$unwind": "$uconn"},
		bson.M{"$unwind": "$uconn.dat"},
		bson.M{"$project": bson.M{
			"host":             1,
			"src_ip":           "$uconn.src",
			"src_network_uuid": "$uconn.src_network_uuid",
			"src_network_name": "$uconn.src_network_name",
			"conns":            "$uconn.dat.count",
			"bytes":            "$uconn.dat.tbytes",
		}},
		// remove duplicate source for each host and sum bytes
		// and connections per blacklisted hostname.
		// we have to do this in parts because network_name
		// may be different between IPs with the same network_uuid
		bson.M{"$group": bson.M{
			"_id": bson.M{
				"host":             "$_id",
				"src_ip":           "$src_ip",
				"src_network_uuid": "$src_network_uuid",
			},
			"src_network_name": bson.M{"$last": "$src_network_name"},
			"conns":            bson.M{"$sum": "$conns"},
			"tbytes":           bson.M{"$sum": "$tbytes"},
		}},
		bson.M{"$project": bson.M{
			"_id":    0,
			"host":   "$_id.host",
			"conns":  1,
			"tbytes": 1,
			"src": bson.M{
				"ip":           "$_id.src_ip",
				"network_uuid": "$_id.src_network_uuid",
				"network_name": "$src_network_name",
			},
		}},

		bson.M{"$group": bson.M{
			"_id":     "$host",
			"conns":   bson.M{"$sum": "$conns"},
			"tbytes":  bson.M{"$sum": "$tbytes"},
			"sources": bson.M{"$addToSet": "$src"},
		}},
		bson.M{"$project": bson.M{
			"_id":         0,
			"host":        "$_id",
			"uconn_count": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$sources", []interface{}{}}}},
			"conn_count":  "$conns",
			"total_bytes": "$tbytes",
			"sources":     1,
		}},
		bson.M{"$sort": bson.M{sort: -1}},
	}

	if !noLimit {
		blHostsQuery = append(blHostsQuery, bson.M{"$limit": limit})
	}

	var blHosts []hostname.AnalysisView

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.HostnamesTable).Pipe(blHostsQuery).AllowDiskUse().All(&blHosts)

	return blHosts, err
}
