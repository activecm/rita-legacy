package commands

import (
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/reporting"
	"github.com/activecm/rita/resources"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{

		Name:  "html-report",
		Usage: "Create an html report for an analyzed database",
		UsageText: "rita html-report [command-options] [database]\n\n" +
			"If no database is specified, a report will be created for every database.",
		Flags: []cli.Flag{
			configFlag,
		},
		Action: func(c *cli.Context) error {
			res := resources.InitResources(c.String("config"))
			databaseName := c.Args().Get(0)
			var databases []database.RITADatabase
			if databaseName != "" {
				ritaDB, err := res.DBIndex.GetDatabase(databaseName)
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				databases = append(databases, ritaDB)
			} else {
				ritaDBs, err := res.DBIndex.GetAnalyzedDatabases()
				if err != nil {
					return cli.NewExitError(err.Error(), -1)
				}
				databases = append(databases, ritaDBs...)
			}
			err := reporting.PrintHTML(databases, res)
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}
