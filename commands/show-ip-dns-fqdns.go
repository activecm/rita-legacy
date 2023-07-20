package commands

import (
	"fmt"
	"os"

	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/resources"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{
		Name:      "show-ip-dns-fqdns",
		Usage:     "Print FQDNs associated with IP via DNS",
		ArgsUsage: "<database> <ip address>",
		Flags: []cli.Flag{
			ConfigFlag,
			humanFlag,
			delimFlag,
		},
		Action: showIpFqdns,
	}

	bootstrapCommands(command)
}

func showIpFqdns(c *cli.Context) error {
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
		err := showIpFqdnsHuman(fqdnResults)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
		return nil
	}

	err = showIpFqdnsRaw(fqdnResults)
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}

	return nil
}

func showIpFqdnsHuman(data []*hostname.FQDNResult) error {
	table := tablewriter.NewWriter(os.Stdout)
	headerFields := []string{
		"Resolved FQDN",
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

func showIpFqdnsRaw(data []*hostname.FQDNResult) error {
	fmt.Println("Resolved FQDN")
	for _, d := range data {
		fmt.Println(d.Hostname)
	}
	return nil
}
