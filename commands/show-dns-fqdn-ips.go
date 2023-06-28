package commands

import (
	"fmt"

	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/resources"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{
		Name:      "show-dns-fqdn-ips",
		Usage:     "Print IPs associated with FQDN via DNS",
		ArgsUsage: "<database> <fqdn>",
		Flags: []cli.Flag{
			ConfigFlag,
			// humanFlag,
			// delimFlag,
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
	// showBeaconsHuman showBeaconsDelim (look at show-beacons.go)
	// AdjustNames (e.g. showFqdnIpsHuman, etc)
	fmt.Println(ipResults)
	
	return nil
}
