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
		return cli.NewExitError(
			"Import failed.\nUse 'rita import' to import the directories "+
				"specified in the config file or 'rita import -i [import-dir] -d [database-name]' "+
				"to import bro logs from a given directory.", -1)
	}

	//both flags were set
	if importDir != "" && databaseName != "" {
		res.Config.S.Bro.LogPath = importDir
		res.Config.S.Bro.DBPrefix = ""
		//Clear out the directory map and set the default database
		res.Config.S.Bro.DirectoryMap = make(map[string]string)
		res.Config.S.Bro.DefaultDatabase = databaseName
	}

	res.Log.Infof("Importing %s\n", res.Config.S.Bro.LogPath)
	fmt.Println("[+] Importing " + res.Config.S.Bro.LogPath)
	importer := parser.NewFSImporter(res, threads, threads)
	datastore := parser.NewMongoDatastore(res.DB.Session, res.MetaDB,
		res.Config.S.Bro.ImportBuffer, res.Log)
	importer.Run(datastore)
	res.Log.Infof("Finished importing %s\n", res.Config.S.Bro.LogPath)
	return nil
}
