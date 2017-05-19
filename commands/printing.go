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
			configFlag,
			cli.StringFlag{
				Name:  "database, d",
				Usage: "Specify which databases to dump, otherwise will import all databases",
				Value: "",
			},
		},
		Action: func(c *cli.Context) error {
			res := database.InitResources("")
			databaseName := c.String("database")
			if databaseName != "" {
				res.System.BroConfig.DBPrefix = ""
				res.System.BroConfig.DefaultDatabase = databaseName
			}
			return printing.Printing(res)
		},
	}
	bootstrapCommands(command)
}
