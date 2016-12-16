package commands

import (
	"github.com/bglebrun/rita/config"
	"github.com/bglebrun/rita/database"
	"github.com/bglebrun/rita/parser"
	"github.com/bglebrun/rita/parser/docwriter"
	"github.com/urfave/cli"
)

func init() {
	importCommand := cli.Command{
		Name:  "import",
		Usage: "import bro logs into the database",
		Flags: []cli.Flag{
			threadFlag,
			configFlag,
		},
		Action: doImport,
	}

	bootstrapCommands(importCommand)
}

// doImport runs the importer
func doImport(c *cli.Context) error {
	conf := config.InitConfig(c.String("config"))
	metadb := database.NewMetaDBHandle(conf)

	dw := docwriter.New(conf, metadb)
	dw.Start(c.Int("threads"))
	parser.NewWatcher(conf, metadb).Run(dw)

	return nil
}
