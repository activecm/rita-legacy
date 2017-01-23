package commands

import (
	"fmt"

	"github.com/bglebrun/rita/config"
	"github.com/bglebrun/rita/database"
	"github.com/bglebrun/rita/parser"
	"github.com/bglebrun/rita/parser/docwriter"
	"github.com/urfave/cli"
)

func init() {
	importCommand := cli.Command{
		Name:  "import",
		Usage: "Import bro logs into the database",
		Flags: []cli.Flag{
			threadFlag,
			configFlag,
			cli.StringFlag{
				Name:  "import-dir, i",
				Usage: "Import bro logs from a `directory` into a database. This overides the config file. Must be used with --database, -d",
				Value: "",
			},
			cli.StringFlag{
				Name:  "database, d",
				Usage: "Store imported bro logs into a database with the given `name`. This overides the config file. Must be used with --import-dir, -i",
				Value: "",
			},
		},
		Action: doImport,
	}

	bootstrapCommands(importCommand)
}

// doImport runs the importer
func doImport(c *cli.Context) error {
	conf := config.InitConfig(c.String("config"))
	metadb := database.NewMetaDBHandle(conf)
	importDir := c.String("import-dir")
	databaseName := c.String("database")

	//one flag was set
	if importDir != "" && databaseName == "" || importDir == "" && databaseName != "" {
		fmt.Println("Import failed.\nUse 'rita import' to import the directories " +
			"specified in the config file or 'rita import -i [import-dir] -d [database-name]' " +
			"to import bro logs from a given directory.")
		return nil
	}

	//both flags were set
	if importDir != "" && databaseName != "" {
		conf.System.BroConfig.LogPath = importDir
		conf.System.BroConfig.DBPrefix = ""
		//Clear out the directory map and set the default database
		conf.System.BroConfig.DirectoryMap = make(map[string]string)
		conf.System.BroConfig.DefaultDatabase = databaseName
	}

	fmt.Printf("Importing %s\n", conf.System.BroConfig.LogPath)
	dw := docwriter.New(conf, metadb)
	dw.Start(c.Int("threads"))
	parser.NewWatcher(conf, metadb).Run(dw)
	fmt.Println("Finished importing!")
	return nil
}
