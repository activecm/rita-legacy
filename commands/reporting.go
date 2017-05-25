package commands

import (
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/reporting"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:  "html-report",
		Usage: "Write scanning information to html output",
		Flags: []cli.Flag{
			configFlag,
			cli.StringFlag{
				Name:  "database, d",
				Usage: "Specify which databases to export, otherwise will export all databases",
				Value: "",
			},
		},
		Action: func(c *cli.Context) error {
			res := database.InitResources("")
			databaseName := c.String("database")
			var databases []string
			if databaseName != "" {
				databases = append(databases, databaseName)
			} else {
				databases = res.MetaDB.GetDatabases()
			}
			return reporting.PrintHTML(databases, res)
		},
	}
	bootstrapCommands(command)
}
