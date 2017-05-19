package commands

import (
	"github.com/bglebrun/rita/database"
	"github.com/bglebrun/rita/printing"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:  "Print-scans",
		Usage: "Write scanning information to html output",
		Flags: []cli.Flag{
			databaseFlag,
		},
		Action: func(c *cli.Context) error {
			if c.String("database") == "" {
				return cli.NewExitError("Specify a database with -d", -1)
			}

			res := database.InitResources("")
			return printing.Printing(res)
		},
	}
	bootstrapCommands(command)
}
