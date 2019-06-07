package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/activecm/rita/parser"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"

	"github.com/urfave/cli"
)

func init() {
	importCommand := cli.Command{
		Name:  "import",
		Usage: "Import bro logs into a target database",
		UsageText: "rita import [command options] [<import directory> <database name>]\n\n" +
			"Logs directly in <import directory> will be imported into a database" +
			" named <database name>.",
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
			fmt.Println(updateCheck(c.String("config")))
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
		return cli.NewExitError("\n\t[!] Both <directory to import> and <database name> are required.", -1)
	}

	// check if import directory is okay to read from
	err := i.checkImportDirExists()
	if err != nil {
		return err
	}

	err = i.checkForInvalidDBChars(i.targetDatabase)
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}

	i.res = resources.InitResources(i.configFile)

	return nil
}

func (i *Importer) checkImportDirExists() error {

	// parse directory path
	filePath, err := filepath.Abs(i.importDir)
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}

	// check if directory exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return cli.NewExitError(fmt.Errorf("\n\t[!] %v", err.Error()), -1)
	}
	return nil
}

func (i *Importer) setRolling() error {
	exists, isRolling, currChunk, totalChunks, err := i.res.MetaDB.GetRollingSettings(i.targetDatabase)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("\n\t[!] Error while reading existing database settings: %v", err.Error()), -1)
	}

	// Can uncomment this check if we want to preserve old behavior with regards to non-rolling databases
	// if exists && !isRolling {
	// 	return cli.NewExitError("\n\t[!] New data cannot be imported into an existing non-rolling database", -1)
	// }

	// a user-provided value for either of the chunk options implies rolling
	if i.totalChunks != -1 || i.currentChunk != -1 {
		i.rolling = true
	}

	if i.totalChunks != -1 { // user gave the total number of chunks via command line
		// use the user-provided value
		i.res.Config.S.Rolling.TotalChunks = i.totalChunks
	} else { // user didn't specify the total number of chunks
		if !exists && !i.rolling {
			// if the database doesn't exist and wasn't specified to be rolling
			// then assume only one chunk
			i.res.Config.S.Rolling.TotalChunks = 1
		} else if exists && isRolling {
			// if the database is already rolling use the existing value
			i.res.Config.S.Rolling.TotalChunks = totalChunks
		} else {
			// otherwise we're converting a non-rolling database or creating a new rolling database
			// and the user didn't specify so use the default value
			i.res.Config.S.Rolling.TotalChunks = i.res.Config.S.Rolling.DefaultChunks
		}
	}

	if i.currentChunk != -1 { // user gave the current chunk via command line
		// use the user-provided value
		i.res.Config.S.Rolling.CurrentChunk = i.currentChunk
	} else { // user didn't specify the current chunk
		if !exists {
			// if the databse doesn't exist, then assume this is the first and only chunk
			i.res.Config.S.Rolling.CurrentChunk = 0
		} else {
			// otherwise increment tne current value, wrapping back to 0 when needed
			i.res.Config.S.Rolling.CurrentChunk = (currChunk + 1) % i.res.Config.S.Rolling.TotalChunks
		}
	}

	// already existing db is converted to rolling if it wasn't already
	// or if the user specified that a new database is rolling
	i.res.Config.S.Rolling.Rolling = exists || i.rolling

	// validate chunk size
	if (i.res.Config.S.Rolling.CurrentChunk < 0 ||
		i.res.Config.S.Rolling.CurrentChunk >= i.res.Config.S.Rolling.TotalChunks) {
		return cli.NewExitError(
			fmt.Sprintf(
				"\n\t[!] Current chunk number [ %d ] must be 0 or greater and less than the total number of chunks [ %d ]",
				i.res.Config.S.Rolling.CurrentChunk,
				i.res.Config.S.Rolling.TotalChunks,
			),
			-1,
		)
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
	i.res.DB.SelectDB(i.targetDatabase)

	// set up rolling stats if they apply
	err = i.setRolling()
	if err != nil {
		return err
	}

	importer := parser.NewFSImporter(i.res, i.threads, i.threads, i.importDir)
	if len(importer.GetInternalSubnets()) == 0 {
		return cli.NewExitError("Internal subnets are not defined. Please set the InternalSubnets section of the config file.", -1)
	}

	i.res.Log.Infof("Importing %s\n", i.importDir)
	fmt.Println("\n\t[+] Importing " + i.importDir + " :")

	importer.Run()

	i.res.Log.Infof("Finished importing %s\n", i.importDir)

	return nil
}

// validates target db name
func (i *Importer) checkForInvalidDBChars(db string) error {
	invalidChars := "/\\.,*<>:|?$#"
	if strings.ContainsAny(db, invalidChars) {
		return fmt.Errorf("\n\t[!] database cannot contain the characters < /, \\, ., \", *, <, >, :, |, ?, $ > as well as spaces or the null character")
	}
	return nil
}
