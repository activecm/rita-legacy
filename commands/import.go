package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/parser"
	"github.com/ocmdev/rita/util"
	"github.com/urfave/cli"
)

func init() {
	importCommand := cli.Command{
		Name:      "import",
		Usage:     "Import bro logs into a database",
		ArgsUsage: "<directory to import> <target database>",
		Flags: []cli.Flag{
			threadFlag,
			configFlag,
			cli.StringFlag{
				Name:  "split, s",
				Usage: "Split the imported bro logs. Accepted values: \"subfolder\", \"date\"",
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
	importDir := c.Args().Get(0)
	targetDatabase := c.Args().Get(1)
	splitStrategy := strings.ToLower(c.String("split"))
	threads := util.Max(c.Int("threads")/2, 1)

	if importDir == "" || targetDatabase == "" {
		return cli.NewExitError("Specify a directory to import and a target database", -1)
	}

	//Remove tailing / when applicable
	if strings.HasSuffix(importDir, string(os.PathSeparator)) {
		importDir = importDir[:len(importDir)-len(string(os.PathSeparator))]
	}

	res.Config.R.Bro.ImportDirectory = importDir
	res.Config.R.Bro.TargetDatabase = targetDatabase
	res.Config.R.Bro.SplitStrategy = config.SplitNone
	if splitStrategy == "subfolder" {
		res.Config.R.Bro.SplitStrategy = config.SplitSubfolder
	} else if splitStrategy == "date" {
		res.Config.R.Bro.SplitStrategy = config.SplitDate
	}

	res.Log.Infof("Importing %s\n", res.Config.R.Bro.ImportDirectory)
	fmt.Println("[+] Importing " + res.Config.R.Bro.ImportDirectory)
	importer := parser.NewFSImporter(res, threads, threads)
	datastore := parser.NewMongoDatastore(res.DB.Session, res.MetaDB,
		res.Config.S.Bro.ImportBuffer, res.Log)
	importer.Run(datastore)
	res.Log.Infof("Finished importing %s\n", res.Config.R.Bro.ImportDirectory)
	return nil
}
