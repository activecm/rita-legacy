package commands

import (
	"fmt"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/parser"
	"github.com/ocmdev/rita/util"
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
	res := database.InitResources(c.String("config"))
	importDir := c.String("import-dir")
	databaseName := c.String("database")
	threads := util.Max(c.Int("threads")/2, 1)

	//one flag was set
	if importDir != "" && databaseName == "" || importDir == "" && databaseName != "" {
		fmt.Println("Import failed.\nUse 'rita import' to import the directories " +
			"specified in the config file or 'rita import -i [import-dir] -d [database-name]' " +
			"to import bro logs from a given directory.")
		return nil
	}

	//both flags were set
	if importDir != "" && databaseName != "" {
		res.System.BroConfig.LogPath = importDir
		res.System.BroConfig.DBPrefix = ""
		//Clear out the directory map and set the default database
		res.System.BroConfig.DirectoryMap = make(map[string]string)
		res.System.BroConfig.DefaultDatabase = databaseName
	}

	fmt.Printf("Importing %s\n", res.System.BroConfig.LogPath)
	importer := parser.NewFSImporter(res, threads, threads)
	datastore := parser.NewMongoDatastore(res.DB.Session, 1000, res.Log)
	importer.Run(datastore)
	fmt.Println("Finished importing!")
	return nil
}
