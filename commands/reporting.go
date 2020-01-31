package commands

import (
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
			dirFlag,
		},
		Action: func(c *cli.Context) error {
			res := resources.InitResources(c.String("config"))
			dir := c.String("directory")
			databaseName := c.Args().Get(0)
			var databases []string
			if databaseName != "" {
				databases = append(databases, databaseName)
			} else {
				databases = res.MetaDB.GetAnalyzedDatabases()
			}
			err := reporting.PrintHTML(databases, dir, res)
			if err != nil {
				return cli.NewExitError(err.Error(), -1)
			}
			return nil
		},
	}
	bootstrapCommands(command)
}
