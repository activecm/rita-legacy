package commands

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/activecm/rita/pkg/blacklist"
	"github.com/activecm/rita/resources"
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
	showNetNames := c.Bool("network-names")
	if db == "" {
		return db, sort, connected, human, showNetNames, cli.NewExitError("Specify a database", -1)
	}
	if sort != "conn_count" && sort != "total_bytes" {
		return db, sort, connected, human, showNetNames, cli.NewExitError("Invalid option passed to sort flag", -1)
	}
	return db, sort, connected, human, showNetNames, nil
}

func printBLSourceIPs(c *cli.Context) error {
	db, sort, connected, human, showNetNames, err := parseBLArgs(c)
	if err != nil {
		return err
	}
	res := resources.InitResources(c.String("config"))
	res.DB.SelectDB(db)

	data, err := blacklist.SrcIPResults(res, sort, c.Int("limit"), c.Bool("no-limit"))

	if err != nil {
		res.Log.Error(err)
		return cli.NewExitError(err, -1)
	}

	if len(data) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if human {
		err = showBLIPsHuman(data, connected, showNetNames, true)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	} else {
		err = showBLIPs(data, connected, showNetNames, true, c.String("delimiter"))
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	}
	return nil
}

func printBLDestIPs(c *cli.Context) error {
	db, sort, connected, human, showNetNames, err := parseBLArgs(c)
	if err != nil {
		return err
	}

	res := resources.InitResources(c.String("config"))
	res.DB.SelectDB(db)

	data, err := blacklist.DstIPResults(res, sort, c.Int("limit"), c.Bool("no-limit"))

	if err != nil {
		res.Log.Error(err)
		return cli.NewExitError(err, -1)
	}

	if len(data) == 0 {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if human {
		err = showBLIPsHuman(data, connected, showNetNames, false)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	} else {
		err = showBLIPs(data, connected, showNetNames, false, c.String("delimiter"))
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	}
	return nil
}

func showBLIPs(ips []blacklist.IPResult, connectedHosts, showNetNames, source bool, delim string) error {
	var headers []string
	if showNetNames {
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
		if showNetNames {
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
				if showNetNames {
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

func showBLIPsHuman(ips []blacklist.IPResult, connectedHosts, showNetNames, source bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	var headers []string
	if showNetNames {
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
		if showNetNames {
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
				if showNetNames {
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
