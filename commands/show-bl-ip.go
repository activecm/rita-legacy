package commands

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/activecm/rita/pkg/blacklist"
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
			delimFlag,
			netNamesFlag,
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
			limitFlag,
			noLimitFlag,
			delimFlag,
			netNamesFlag,
		},
		Usage:  "Print blacklisted IPs which received connections",
		Action: printBLDestIPs,
	}

	bootstrapCommands(blSourceIPs, blDestIPs)
}

func parseBLArgs(c *cli.Context) (string, string, bool, bool, bool, error) {
	db := c.Args().Get(0)
	sort := c.String("sort")
	connected := c.Bool("connected")
	human := c.Bool("human-readable")
	netNames := c.Bool("network-names")
	if db == "" {
		return db, sort, connected, human, netNames, cli.NewExitError("Specify a database", -1)
	}
	if sort != "conn_count" && sort != "total_bytes" {
		return db, sort, connected, human, netNames, cli.NewExitError("Invalid option passed to sort flag", -1)
	}
	return db, sort, connected, human, netNames, nil
}

func printBLSourceIPs(c *cli.Context) error {
	db, sort, connected, human, netNames, err := parseBLArgs(c)
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

	data, err := getBlacklistedIPsResultsView(res, sort, c.Bool("no-limit"), c.Int("limit"), match, "src", "dst")

	if err != nil {
		res.Log.Error(err)
		return cli.NewExitError(err, -1)
	}

	if len(data) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if human {
		err = showBLIPsHuman(data, connected, netNames, true)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	} else {
		err = showBLIPs(data, connected, netNames, true, c.String("delimiter"))
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	}
	return nil
}

func printBLDestIPs(c *cli.Context) error {
	db, sort, connected, human, netNames, err := parseBLArgs(c)
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

	data, err := getBlacklistedIPsResultsView(res, sort, c.Bool("no-limit"), c.Int("limit"), match, "dst", "src")

	if err != nil {
		res.Log.Error(err)
		return cli.NewExitError(err, -1)
	}

	if len(data) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if human {
		err = showBLIPsHuman(data, connected, netNames, false)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	} else {
		err = showBLIPs(data, connected, netNames, false, c.String("delimiter"))
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	}
	return nil
}

func showBLIPs(ips []blacklist.ResultsView, connectedHosts, netNames, source bool, delim string) error {
	var headers []string
	if netNames {
		headers = []string{"IP", "Network"}
	} else {
		headers = []string{"IP"}
	}

	headers = append(headers, "Connections", "Unique Connections", "Total Bytes")

	if connectedHosts {
		if source {
			headers = append(headers, "Destinations")
		} else {
			headers = append(headers, "Sources")
		}
	}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headers, delim))
	for _, entry := range ips {

		var serialized []string
		if netNames {
			serialized = []string{entry.Host.IP, entry.Host.NetworkName}
		} else {
			serialized = []string{entry.Host.IP}
		}

		serialized = append(serialized,
			strconv.Itoa(entry.Connections),
			strconv.Itoa(entry.UniqueConnections),
			strconv.Itoa(entry.TotalBytes),
		)

		if connectedHosts {
			var connectedHostsIPs []string
			for _, connectedUniqIP := range entry.Peers {

				var connectedIPStr string
				if netNames {
					escapedNetName := strings.ReplaceAll(connectedUniqIP.NetworkName, " ", "_")
					escapedNetName = strings.ReplaceAll(escapedNetName, ":", "_")
					connectedIPStr = escapedNetName + ":" + connectedUniqIP.IP
				} else {
					connectedIPStr = connectedUniqIP.IP
				}

				connectedHostsIPs = append(connectedHostsIPs, connectedIPStr)
			}
			sort.Strings(connectedHostsIPs)
			serialized = append(serialized, strings.Join(connectedHostsIPs, " "))
		}
		fmt.Println(
			strings.Join(
				serialized,
				delim,
			),
		)
	}
	return nil
}

func showBLIPsHuman(ips []blacklist.ResultsView, connectedHosts, netNames, source bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	var headers []string
	if netNames {
		headers = []string{"IP", "Network"}
	} else {
		headers = []string{"IP"}
	}

	headers = append(headers, "Connections", "Unique Connections", "Total Bytes")

	if connectedHosts {
		if source {
			headers = append(headers, "Destinations")
		} else {
			headers = append(headers, "Sources")
		}
	}
	table.SetHeader(headers)
	for _, entry := range ips {

		var serialized []string
		if netNames {
			serialized = []string{entry.Host.IP, entry.Host.NetworkName}
		} else {
			serialized = []string{entry.Host.IP}
		}

		serialized = append(serialized,
			strconv.Itoa(entry.Connections),
			strconv.Itoa(entry.UniqueConnections),
			strconv.Itoa(entry.TotalBytes),
		)

		if connectedHosts {
			var connectedHostsIPs []string
			for _, connectedUniqIP := range entry.Peers {

				var connectedIPStr string
				if netNames {
					escapedNetName := strings.ReplaceAll(connectedUniqIP.NetworkName, " ", "_")
					escapedNetName = strings.ReplaceAll(escapedNetName, ":", "_")
					connectedIPStr = escapedNetName + ":" + connectedUniqIP.IP
				} else {
					connectedIPStr = connectedUniqIP.IP
				}

				connectedHostsIPs = append(connectedHostsIPs, connectedIPStr)
			}
			sort.Strings(connectedHostsIPs)
			serialized = append(serialized, strings.Join(connectedHostsIPs, " "))
		}
		table.Append(serialized)
	}
	table.Render()
	return nil
}

//getBlaclistedIPsResultsView
func getBlacklistedIPsResultsView(res *resources.Resources, sort string, noLimit bool, limit int, match bson.M, field1 string, field2 string) ([]blacklist.ResultsView, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var blIPs []blacklist.ResultsView

	blIPQuery := []bson.M{
		// find blacklisted source/ destination hosts
		bson.M{"$match": match},
		// only select ip info from hosts collection
		bson.M{"$project": bson.M{
			"ip":           1,
			"network_uuid": 1,
			"network_name": 1,
		}},
		// ideally, we'd join on both src/dst and src/dst_network_uuid, but MongoDB
		// doesn't allow multi-field joins in versions < 3.6
		// so, we will join on the ip and check the uuids manually next.
		// find uconns where this ip appears as src/dst
		bson.M{"$lookup": bson.M{
			"from":         "uconn",
			"localField":   "ip",
			"foreignField": field1,
			"as":           "uconn",
		}},
		// convert lookup array to separate records
		bson.M{"$unwind": "$uconn"},
		// check if the network uuids match each other
		bson.M{"$project": bson.M{
			"ip":           1,
			"network_uuid": 1,
			"network_name": 1,
			"uconn":        1,
			"match_uuid": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []string{"$network_uuid", "$uconn." + field1 + "_network_uuid"}},
					1, 0,
				},
			},
		}},
		// drop the records which do not have matching uuids
		bson.M{"$match": bson.M{
			"match_uuid": 1,
		}},
		// start aggregation across chunks/ time
		bson.M{"$unwind": "$uconn.dat"},
		// simplify names/ drop unused data
		bson.M{"$project": bson.M{
			"ip":                1,
			"network_uuid":      1,
			"network_name":      1,
			"peer_ip":           "$uconn." + field2,
			"peer_network_uuid": "$uconn." + field2 + "_network_uuid",
			"peer_network_name": "$uconn." + field2 + "_network_name",
			"conns":             "$uconn.dat.count",
			"tbytes":            "$uconn.dat.tbytes",
		}},
		// we want to group on the blacklisted IP we started with and find
		// the set of its peer IPs. Creating a set over {ip, network_uuid, network_name} objects
		// takes a bit of work since the network_name may change over time and should not
		// be used when determining equality.
		// to get around this, we pick one of the names associated with a
		// given peer uuid and throw away the rest.
		// aggregate uconn data through time (over chunks)
		bson.M{"$group": bson.M{
			"_id": bson.M{
				// group within each blacklisted host
				"ip":           "$ip",
				"network_uuid": "$network_uuid",
				// group on the peers which connected to the blacklisted host
				"peer_ip":           "$peer_ip",
				"peer_network_uuid": "$peer_network_uuid",
			},
			// there should only be one network_name in each record
			// as it comes from the hosts collection
			"network_name": bson.M{"$last": "$network_name"},
			// use one of the network names associated with the network_uuid
			// for this partial result
			"peer_network_name": bson.M{"$last": "$peer_network_name"},
			// compute the partial sums over connections and bytes
			"conns":  bson.M{"$sum": "$conns"},
			"tbytes": bson.M{"$sum": "$tbytes"},
		}},
		// gather the peer fields so we can use addToSet
		bson.M{"$project": bson.M{
			"_id":          0, //move the id fields back out
			"ip":           "$_id.ip",
			"network_uuid": "$_id.network_uuid",
			"network_name": "$network_name",
			"peer": bson.M{
				"ip":           "$_id.peer_ip",
				"network_uuid": "$_id.peer_network_uuid",
				"network_name": "$peer_network_name",
			},
			"conns":  1,
			"tbytes": 1,
		}},
		// group the uconn data up to find which IPs peered with this blacklisted host,
		// how many connections were made, and how much data was sent in total.
		{"$group": bson.M{
			"_id": bson.M{
				"ip":           "$ip",
				"network_uuid": "$network_uuid",
				"network_name": "$network_name",
			},
			"peers":  bson.M{"$addToSet": "$peer"},
			"conns":  bson.M{"$sum": "$conns"},
			"tbytes": bson.M{"$sum": "$tbytes"},
		}},
		// move the id fields back out and add uconn_count
		bson.M{"$project": bson.M{
			"_id":          0,
			"ip":           "$_id.ip",
			"network_uuid": "$_id.network_uuid",
			"network_name": "$_id.network_name",
			"peers":        1,
			"conn_count":   "$conns",
			"uconn_count":  bson.M{"$size": bson.M{"$ifNull": []interface{}{"$peers", []interface{}{}}}},
			"total_bytes":  "$tbytes",
		}},
		bson.M{"$sort": bson.M{sort: -1}},
	}

	if !noLimit {
		blIPQuery = append(blIPQuery, bson.M{"$limit": limit})
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.HostTable).Pipe(blIPQuery).AllowDiskUse().All(&blIPs)

	return blIPs, err

}
