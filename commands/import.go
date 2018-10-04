package commands

import (
	"fmt"
	"path/filepath"

	"github.com/activecm/rita/parser"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/urfave/cli"
)

func init() {
	importCommand := cli.Command{
		Name:  "import",
		Usage: "Import bro logs into a target database",
		UsageText: "rita import [command options] [<import directory> <database root>]\n\n" +
			"Logs directly in <import directory> will be imported into a database" +
			" named <database root>. Files in a subfolder of <import directory> will be imported" +
			" into <database root>-$SUBFOLDER_NAME. <import directory>" +
			" and <database root> will be loaded from the configuration file unless" +
			" BOTH arguments are supplied.",
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
	res := resources.InitResources(c.String("config"))
	importDir := c.Args().Get(0)
	targetDatabase := c.Args().Get(1)
	threads := util.Max(c.Int("threads")/2, 1)

	//check if one argument is set but not the other
	if importDir != "" && targetDatabase == "" ||
		importDir == "" && targetDatabase != "" {
		return cli.NewExitError("Both <directory to import> and <database prefix> are required to override the config file.", -1)
	}

	//check if the user overrode the config file
	if importDir != "" && targetDatabase != "" {
		//expand relative path
		//nolint: vetshadow
		importDir, err := filepath.Abs(importDir)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}

		res.Config.S.Bro.ImportDirectory = importDir
		res.Config.S.Bro.DBRoot = targetDatabase
	}

	res.Log.Infof("Importing %s\n", res.Config.S.Bro.ImportDirectory)
	fmt.Println("[+] Importing " + res.Config.S.Bro.ImportDirectory)
	importer := parser.NewFSImporter(res, threads, threads)
	datastore, err := parser.NewMongoDatastore(
		res.DB.Session,
		res.DBIndex,
		res.Config.S.Bro.ImportBuffer,
		res.Config.R.Version,
		res.Log,
	)
	if err != nil {
		return cli.NewExitError("Error: could not set up connection with MongoDB: "+err.Error(), -1)
	}
	err = importer.Run(datastore)
	if err != nil {
		return err
	}
	res.Log.Infof("Finished importing %s\n", res.Config.S.Bro.ImportDirectory)
	return nil
}
