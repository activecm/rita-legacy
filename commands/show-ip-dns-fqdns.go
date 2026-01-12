package commands

import (
	"fmt"
	"os"

	"github.com/activecm/rita-legacy/pkg/hostname"
	"github.com/activecm/rita-legacy/resources"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{
		Name:      "show-ip-dns-fqdns",
		Usage:     "Print FQDNs associated with IP Address via DNS",
		ArgsUsage: "<database> <ip address>",
		Flags: []cli.Flag{
			ConfigFlag,
			humanFlag,
			delimFlag,
		},
		Action: showIPFqdns,
	}

	bootstrapCommands(command)
}

func showIPFqdns(c *cli.Context) error {
	db, ip := c.Args().Get(0), c.Args().Get(1)
	if db == "" || ip == "" {
		return cli.NewExitError("Specify a database and IP address", -1)
	}

	res := resources.InitResources(getConfigFilePath(c))
	res.DB.SelectDB(db)

	fqdnResults, err := hostname.FQDNResults(res, ip)

	if err != nil {
		res.Log.Error(err)
		return cli.NewExitError(err, -1)
	}

	if !(len(fqdnResults) > 0) {
		return cli.NewExitError("No results were found for "+db, -1)
	}

	if c.Bool("human-readable") {
		err := showIPFqdnsHuman(fqdnResults)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
		return nil
	}

	err = showIPFqdnsRaw(fqdnResults)
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}

	return nil
}

func showIPFqdnsHuman(data []*hostname.FQDNResult) error {
	table := tablewriter.NewWriter(os.Stdout)
	headerFields := []string{
		"Queried FQDN",
	}

	table.SetHeader(headerFields)

	for _, d := range data {
		row := []string{
			d.Hostname,
		}
		table.Append(row)
	}
	table.Render()
	return nil
}

func showIPFqdnsRaw(data []*hostname.FQDNResult) error {
	fmt.Println("Queried FQDN")
	for _, d := range data {
		fmt.Println(d.Hostname)
	}
	return nil
}
