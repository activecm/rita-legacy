package commands

import (
	"github.com/activecm/rita/resources"
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
			netNamesFlag,
		},
		Action: showIpFqdns,
	}

	bootstrapCommands(command)
}

func showIpFqdns(c *cli.Context) error {
	db := c.Args().Get(0)
	if db == "" {
		return cli.NewExitError("Specify a database", -1)
	}

	ip := c.Args().Get(1)
	if ip == "" {
		return cli.NewExitError("Specify an IP address", -1)
	}
	res := resources.InitResources(getConfigFilePath(c))
	res.DB.SelectDB(db)

	return nil
}