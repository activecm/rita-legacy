package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/activecm/rita-legacy/pkg/data"
	"github.com/activecm/rita-legacy/pkg/hostname"
	"github.com/activecm/rita-legacy/resources"
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
			netNamesFlag,
		},
		Action: showFqdnIps,
	}

	bootstrapCommands(command)
}

func showFqdnIps(c *cli.Context) error {
	db, fqdn := c.Args().Get(0), c.Args().Get(1)
	if db == "" || fqdn == "" {
		return cli.NewExitError("Specify a database and FQDN", -1)
	}
	res := resources.InitResources(getConfigFilePath(c))
	res.DB.SelectDB(db)

	ipResults, err := hostname.IPResults(res, fqdn)

	if err != nil {
		res.Log.Error(err)
		return cli.NewExitError(err, -1)
	}

	if !(len(ipResults) > 0) {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	showNetNames := c.Bool("network-names")

	if c.Bool("human-readable") {
		err := showFqdnIpsHuman(ipResults, showNetNames)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
		return nil
	}

	err = showFqdnIpsDelim(ipResults, c.String("delimiter"), showNetNames)
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}

	return nil
}

func showFqdnIpsHuman(data []data.UniqueIP, showNetNames bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	var headerFields []string
	if showNetNames {
		headerFields = []string{
			"Resolved IP", "Network",
		}
	} else {
		headerFields = []string{
			"Resolved IP",
		}
	}

	table.SetHeader(headerFields)

	for _, d := range data {
		var row []string
		if showNetNames {
			row = []string{
				d.IP, d.NetworkName,
			}
		} else {
			row = []string{
				d.IP,
			}
		}
		table.Append(row)
	}
	table.Render()
	return nil
}

func showFqdnIpsDelim(data []data.UniqueIP, delim string, showNetNames bool) error {
	var headerFields []string
	if showNetNames {
		headerFields = []string{
			"Resolved IP", "Network",
		}
	} else {
		headerFields = []string{
			"Resolved IP",
		}
	}

	// Print the headers and analytic values, separated by a delimiter
	fmt.Println(strings.Join(headerFields, delim))
	for _, d := range data {
		var row []string
		if showNetNames {
			row = []string{
				d.IP, d.NetworkName,
			}
		} else {
			row = []string{
				d.IP,
			}
		}

		fmt.Println(strings.Join(row, delim))
	}
	return nil
}
