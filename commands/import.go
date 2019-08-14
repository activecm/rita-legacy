package commands

import (
	"fmt"
	"strings"

	"github.com/activecm/rita/parser"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/activecm/rita/config"

	"github.com/urfave/cli"
)

func init() {
	importCommand := cli.Command{
		Name:  "import",
		Usage: "Import bro logs into a target database",
		UsageText: "rita import [command options] <import directory|file> [<import directory|file>...] <database name>\n\n" +
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
		args           cli.Args
		importFiles    []string
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
		args:           c.Args(),
		rolling:        c.Bool("rolling"),
		totalChunks:    c.Int("numchunks"),
		currentChunk:   c.Int("chunk"),
		threads:        util.Max(c.Int("threads")/2, 1),
	}
}

func (i *Importer) parseArgs() error {
	if len(i.args) < 2 {
		return cli.NewExitError("\n\t[!] Both <files/directory to import> and <database name> are required.", -1)
	}

	i.targetDatabase = i.args[len(i.args)-1]  // the last argument
	i.importFiles = i.args[:len(i.args)-1]    // all except the last argument

	//check if one argument is set but not the other
	if i.importFiles[0] == "" || i.targetDatabase == "" {
		return cli.NewExitError("\n\t[!] Both <files/directory to import> and <database name> are required.", -1)
	}

	// check if import directory is okay to read from
	err := checkFilesExist(i.importFiles)
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

func checkFilesExist(files []string) error {
	for _, file := range files {
		if !util.Exists(file) {
			return cli.NewExitError(fmt.Errorf("\n\t[!] %v cannot be found", file), -1)
		}
	}
	return nil
}

// setRolling determines what the current rolling and chunk settings should be
// based on the values already in the database and the values supplied on the
// command line by the user.
func setRolling(dbExists bool, dbIsRolling bool, dbCurrChunk int, dbTotalChunks int,
	userIsRolling bool, userCurrChunk int, userTotalChunks int, cfgDefaultChunks int) (config.RollingStaticCfg, error) {

	cfg := config.RollingStaticCfg{}

	// Can uncomment this check if we want to preserve old behavior with regards to non-rolling databases
	// if dbExists && !dbIsRolling {
	// 	return cfg, errors.New("\n\t[!] New data cannot be imported into an existing non-rolling database")
	// }

	// a user-provided value for either of the chunk options implies rolling
	if userTotalChunks != -1 || userCurrChunk != -1 {
		userIsRolling = true
	}

	if userTotalChunks != -1 { // user gave the total number of chunks via command line
		// it's currently an error to try to reduce the total number of chunks in an existing rolling database
		if dbExists && dbIsRolling && userTotalChunks < dbTotalChunks  {
			return cfg, fmt.Errorf(
				"\n\t[!] Cannot modify the total number of chunks in an existing database [ %d ]",
				dbTotalChunks,
			)
		}

		// use the user-provided value
		cfg.TotalChunks = userTotalChunks
	} else { // user didn't specify the total number of chunks
		if !dbExists && !userIsRolling {
			// if the database doesn't exist and wasn't specified to be rolling
			// then assume only one chunk
			cfg.TotalChunks = 1
		} else if dbExists && dbIsRolling {
			// if the database is already rolling use the existing value
			cfg.TotalChunks = dbTotalChunks
		} else {
			// otherwise we're converting a non-rolling database or creating a new rolling database
			// and the user didn't specify so use the default value
			cfg.TotalChunks = cfgDefaultChunks
		}
	}

	if userCurrChunk != -1 { // user gave the current chunk via command line
		// use the user-provided value
		cfg.CurrentChunk = userCurrChunk
	} else { // user didn't specify the current chunk
		if !dbExists {
			// if the databse doesn't exist, then assume this is the first chunk
			cfg.CurrentChunk = 0
		} else {
			// otherwise increment tne current value, wrapping back to 0 when needed
			cfg.CurrentChunk = (dbCurrChunk + 1) % cfg.TotalChunks
		}
	}

	// already existing db is converted to rolling if it wasn't already
	// or if the user specified that a new database is rolling
	cfg.Rolling = dbExists || userIsRolling

	// validate chunk size
	if (cfg.CurrentChunk < 0 ||
		cfg.CurrentChunk >= cfg.TotalChunks) {
		return cfg, fmt.Errorf(
			"\n\t[!] Current chunk number [ %d ] must be 0 or greater and less than the total number of chunks [ %d ]",
			cfg.CurrentChunk,
			cfg.TotalChunks,
		)
	}

	return cfg, nil
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

	// get settings from an existing database
	exists, isRolling, currChunk, totalChunks, err := i.res.MetaDB.GetRollingSettings(i.targetDatabase)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("\n\t[!] Error while reading existing database settings: %v", err.Error()), -1)
	}
	// determine the new rolling settings based on current and supplied arguments
	rollingCfg, err := setRolling(exists, isRolling, currChunk, totalChunks,
		i.rolling, i.currentChunk, i.totalChunks, i.res.Config.S.Rolling.DefaultChunks)
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}
	// preserve the default chunks setting (even though we don't use it after this currently)
	rollingCfg.DefaultChunks = i.res.Config.S.Rolling.DefaultChunks
	i.res.Config.S.Rolling = rollingCfg

	importer := parser.NewFSImporter(i.res, i.threads, i.threads, i.importFiles)
	if len(importer.GetInternalSubnets()) == 0 {
		return cli.NewExitError("Internal subnets are not defined. Please set the InternalSubnets section of the config file.", -1)
	}

	i.res.Log.Infof("Importing %v\n", i.importFiles)
	fmt.Printf("\n\t[+] Importing %v:\n", i.importFiles)

	// print out a message if we're automatically converting this to a rolling database
	// i.e. it wasn't rolling before and the user didn't specify the --rolling flag
	if !isRolling && !i.rolling && rollingCfg.Rolling {
		i.res.Log.Infof("Non-rolling database %v will be converted to rolling\n", i.targetDatabase)
		fmt.Printf("\t[+] Non-rolling database %v will be converted to rolling\n", i.targetDatabase)
	}

	importer.Run()

	i.res.Log.Infof("Finished importing %v\n", i.importFiles)

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
