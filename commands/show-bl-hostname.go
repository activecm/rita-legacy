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

	blHostnames := cli.Command{
		Name:      "show-bl-hostnames",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			ConfigFlag,
			humanFlag,
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

	res := resources.InitResources(getConfigFilePath(c))
	res.DB.SelectDB(db)

	data, err := blacklist.HostnameResults(res, "conn_count", c.Int("limit"), c.Bool("no-limit"))

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

func showBLHostnames(hostnames []blacklist.HostnameResult, delim string, showNetNames bool) error {
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
		if showNetNames {
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

func showBLHostnamesHuman(hostnames []blacklist.HostnameResult, showNetNames bool) error {
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
		if showNetNames {
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
