package commands

import (
	"fmt"
	"os"
	"strings"

	// "github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/resources"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{
		Name:      "show-dns-fqdn-ips",
		Usage:     "Print IPs associated with FQDN via DNS",
		ArgsUsage: "<database> <fqdn>",
		Flags: []cli.Flag{
			ConfigFlag,
			humanFlag,
			delimFlag,
			// netNamesFlag,
		},
		Action: showFqdnIps,
	}

	bootstrapCommands(command)
}

func showFqdnIps(c *cli.Context) error {
	db := c.Args().Get(0)
	if db == "" {
		return cli.NewExitError("Specify a database", -1)
	}
	fqdn := c.Args().Get(1)
	res := resources.InitResources(getConfigFilePath(c))
	res.DB.SelectDB(db)

	ipResults, err := hostname.HostnameIPResults(res, fqdn)

	if err != nil {
		return cli.NewExitError(err, -1)
	}

	if c.Bool("human-readable") {
		err := showFqdnIpsHuman(ipResults)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
		return nil
	}

	err = showFqdnIpsDelim(ipResults, c.String("delimiter"))
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}

	return nil
}

func showFqdnIpsHuman(data []data.UniqueIP) error {
	table := tablewriter.NewWriter(os.Stdout)
	headerFields := []string{
		"Source IP", "Network UUID", "Network Name",
	}

	table.SetHeader(headerFields)

	for _, d := range data {
		row := []string{
			d.IP, b(d.NetworkUUID), d.NetworkName,
		}
		table.Append(row)
	}
	table.Render()
	return nil
}

func showFqdnIpsDelim(data []data.UniqueIP, delim string) error {
	headerFields := []string{
		"Source IP", "Network UUID", "Network Name",
	}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headerFields, delim))
	for _, d := range data {
		row := []string{
			d.IP, b(d.NetworkUUID), d.NetworkName,
		}

		fmt.Println(strings.Join(row, delim))
	}
	return nil
}
