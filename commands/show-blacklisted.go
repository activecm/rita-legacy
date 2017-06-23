package commands

import (
	"github.com/ocmdev/rita/database"
	"github.com/urfave/cli"
)

func init() {

	blSourceIPs := cli.Command{
		Name: "show-bl-source-ips",
		Flags: []cli.Flag{
			humanFlag,
			configFlag,
		},
		Usage:  "Print blacklisted IPs which initiated connections",
		Action: printBLSourceIPs,
	}

	blDestIPs := cli.Command{
		Name:  "show-bl-dest-ips",
		Usage: "Print blacklisted IPs which recieved connections",
	}

	blHostnames := cli.Command{
		Name:  "show-bl-hostnames",
		Usage: "Print blacklisted hostnames which recieved connections",
	}

	blURLs := cli.Command{
		Name:  "show-bl-urls",
		Usage: "Print blacklisted URLs which were visited",
	}

	bootstrapCommands(blSourceIPs, blDestIPs, blHostnames, blURLs)
}

func printBLSourceIPs(c *cli.Context) error {
	res := database.InitResources(c.String("config"))
	_ = res
	return nil
}

func printBLDestIPs(c *cli.Context) error {
	res := database.InitResources(c.String("config"))
	_ = res
	return nil
}

func printBLHostnames(c *cli.Context) error {
	res := database.InitResources(c.String("config"))
	_ = res
	return nil
}

func printBLURLs(c *cli.Context) error {
	res := database.InitResources(c.String("config"))
	_ = res
	return nil
}
