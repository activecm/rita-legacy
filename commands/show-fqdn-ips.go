package commands

import (
	"fmt"

	"github.com/urfave/cli"
)

func init() {
	command := cli.Command {
		Name: "show-fqdn-ips",
		Usage: "Print IPs associated with FQDN",
		ArgsUsage: "<fqdn>",
		Action: showFqdnIps,
	}

	bootstrapCommands(command)
}

func showFqdnIps(c *cli.Context) {
	fqdn := c.Args().Get(0)

	fmt.Println(fqdn)
}