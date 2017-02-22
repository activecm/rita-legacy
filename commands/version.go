package commands

import (
	"fmt"
	"github.com/ocmdev/rita/database"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{
		Name:   "version",
		Usage:  "Show rita version",
		Action: showVersion,
	}

	bootstrapCommands(command)
}

func showVersion(c *cli.Context) error {
	res := database.InitResources("")
	fmt.Println(res.System.Version)
	return nil
}
