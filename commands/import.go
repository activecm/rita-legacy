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
			rollingFlag,
			totalChunksFlag,
			currentChunkFlag,
		},
		Action: func(c *cli.Context) error {
			importer := NewImporter(c)
			err := importer.run()
			fmt.Printf(updateCheck(c.String("config")))
			return err
		},
	}

	bootstrapCommands(importCommand)
}

type (
	//Importer ...
	Importer struct {
		res            *resources.Resources
		configFile     string
		importDir      string
		targetDatabase string
		rolling        bool
		totalChunks    int
		currentChunk   int
		threads        int
	}
)

//NewImporter ....
func NewImporter(c *cli.Context) *Importer {
	return &Importer{
		configFile:     c.String("config"),
		importDir:      c.Args().Get(0),
		targetDatabase: c.Args().Get(1),
		rolling:        c.Bool("rolling"),
		totalChunks:    c.Int("numchunks"),
		currentChunk:   c.Int("chunk"),
		threads:        util.Max(c.Int("threads")/2, 1),
	}
}

func (i *Importer) parseArgs() error {

	//check if one argument is set but not the other
	if i.importDir == "" || i.targetDatabase == "" {
		return cli.NewExitError("Both <directory to import> and <database prefix> are required.", -1)
	}

	if i.rolling {
		if (i.totalChunks == -1) || (i.currentChunk == -1) {
			return cli.NewExitError("Both `--numchunks <total number of chunks>` and `--chunk <current chunk number>` must be provided for rolling analysis import.", -1)
		}
		if !(i.totalChunks > 0) {
			return cli.NewExitError("Total number of chunks must be between 1 and 24", -1)
		}
		if !(i.currentChunk > 0) || (i.currentChunk > i.totalChunks) {
			return cli.NewExitError("Current chunk number must be <= (total number of chunks)", -1)
		}
	}

	i.res = resources.InitResources(i.configFile)

	return nil
}

func (i *Importer) setTargetDatabase() error {
	// get all database names
	names, _ := i.res.DB.Session.DatabaseNames()

	// check if database exists
	dbExists := util.StringInSlice(i.targetDatabase, names)

	// Add new metadatabase record for db if doesn't already exist
	if !dbExists {
		err := i.res.MetaDB.AddNewDB(i.targetDatabase)
		if err != nil {
			return cli.NewExitError(err.Error(), -1)
		}
	}

	i.res.DB.SelectDB(i.targetDatabase)
	i.res.Config.S.Bro.DBRoot = i.targetDatabase

	return nil
}

func (i *Importer) setImportDirectory() error {
	var err error
	i.importDir, err = filepath.Abs(i.importDir)
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}
	i.res.Config.S.Bro.ImportDirectory = i.importDir
	return nil
}

func (i *Importer) setRolling() error {
	if i.rolling {
		i.res.Config.S.Bro.Rolling = true
		i.res.Config.S.Bro.TotalChunks = i.totalChunks
		i.res.Config.S.Bro.CurrentChunk = i.currentChunk - 1
	} else {
		i.res.Config.S.Bro.Rolling = false
		i.res.Config.S.Bro.TotalChunks = 1
		i.res.Config.S.Bro.CurrentChunk = 0
	}

	return nil
}

// run runs the importer
func (i *Importer) run() error {
	// verify command line arguments
	err := i.parseArgs()
	if err != nil {
		return err
	}

	// set up target database
	err = i.setTargetDatabase()
	if err != nil {
		return err
	}

	err = i.setImportDirectory()
	if err != nil {
		return err
	}

	err = i.setRolling()
	if err != nil {
		return err
	}

	i.res.Log.Infof("Importing %s\n", i.res.Config.S.Bro.ImportDirectory)
	fmt.Println("[+] Importing " + i.res.Config.S.Bro.ImportDirectory)

	importer := parser.NewFSImporter(i.res, i.threads, i.threads)
	datastore := parser.NewMongoDatastore(i.res.DB.Session, i.res.MetaDB,
		i.res.Config.S.Bro.ImportBuffer, i.res.Log)

	importer.Run(datastore)

	i.res.Log.Infof("Finished importing %s\n", i.res.Config.S.Bro.ImportDirectory)

	return nil
}
