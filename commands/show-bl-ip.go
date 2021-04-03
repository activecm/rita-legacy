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
			ConfigFlag,
			humanFlag,
			blConnFlag,
			blSortFlag,
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
			ConfigFlag,
			humanFlag,
			blConnFlag,
			blSortFlag,
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
	var err error
	if db == "" {
		err = cli.NewExitError("Specify a database", -1)
	} else if sort != "conn_count" && sort != "total_bytes" {
		err = cli.NewExitError("Invalid option passed to sort flag", -1)
	}
	return db, sort, connected, human, showNetNames, err
}

func printBLSourceIPs(c *cli.Context) error {
	db, sort, connected, human, showNetNames, err := parseBLArgs(c)
	if err != nil {
		return err
	}
	res := resources.InitResources(getConfigFilePath(c))
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

	res := resources.InitResources(getConfigFilePath(c))
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
	var headerFields []string
	if !showNetNames && !connectedHosts {
		headerFields = []string{"IP", "Connections", "Unique Connections", "Total Bytes"}
	} else if showNetNames && !connectedHosts {
		headerFields = []string{"IP", "Network", "Connections", "Unique Connections", "Total Bytes"}
	} else if !showNetNames && connectedHosts && source {
		headerFields = []string{"IP", "Connections", "Unique Connections", "Total Bytes", "Destinations"}
	} else if !showNetNames && connectedHosts && !source {
		headerFields = []string{"IP", "Connections", "Unique Connections", "Total Bytes", "Sources"}
	} else if showNetNames && connectedHosts && source {
		headerFields = []string{"IP", "Network", "Connections", "Unique Connections", "Total Bytes", "Destinations"}
	} else if showNetNames && connectedHosts && !source {
		headerFields = []string{"IP", "Network", "Connections", "Unique Connections", "Total Bytes", "Sources"}
	}

	// Print the headerFields and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headerFields, delim))
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
			if showNetNames {
				for _, connectedUniqIP := range entry.Peers {
					escapedNetName := strings.ReplaceAll(connectedUniqIP.NetworkName, " ", "_")
					escapedNetName = strings.ReplaceAll(escapedNetName, ":", "_")
					connectedIPStr := escapedNetName + ":" + connectedUniqIP.IP
					connectedHostsIPs = append(connectedHostsIPs, connectedIPStr)
				}
			} else {
				for _, connectedUniqIP := range entry.Peers {
					connectedHostsIPs = append(connectedHostsIPs, connectedUniqIP.IP)
				}
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
	var headerFields []string

	if !showNetNames && !connectedHosts {
		headerFields = []string{"IP", "Connections", "Unique Connections", "Total Bytes"}
	} else if showNetNames && !connectedHosts {
		headerFields = []string{"IP", "Network", "Connections", "Unique Connections", "Total Bytes"}
	} else if !showNetNames && connectedHosts && source {
		headerFields = []string{"IP", "Connections", "Unique Connections", "Total Bytes", "Destinations"}
	} else if !showNetNames && connectedHosts && !source {
		headerFields = []string{"IP", "Connections", "Unique Connections", "Total Bytes", "Sources"}
	} else if showNetNames && connectedHosts && source {
		headerFields = []string{"IP", "Network", "Connections", "Unique Connections", "Total Bytes", "Destinations"}
	} else if showNetNames && connectedHosts && !source {
		headerFields = []string{"IP", "Network", "Connections", "Unique Connections", "Total Bytes", "Sources"}
	}

	table.SetHeader(headerFields)
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
