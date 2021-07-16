package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/parser"
	"github.com/activecm/rita/pkg/remover"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func init() {
	importCommand := cli.Command{
		Name:  "import",
		Usage: "Import zeek logs into a target database",
		UsageText: "rita import [command options] <import directory|file> [<import directory|file>...] <database name>\n\n" +
			"Logs directly in <import directory> will be imported into a database" +
			" named <database name>.",
		Flags: []cli.Flag{
			ConfigFlag,
			threadFlag,
			deleteFlag,
			rollingFlag,
			totalChunksFlag,
			currentChunkFlag,
		},
		Action: func(c *cli.Context) error {
			importer := NewImporter(c)
			err := importer.run()
			fmt.Println(updateCheck(getConfigFilePath(c)))
			return err
		},
	}

	bootstrapCommands(importCommand)
}

type (
	//Importer ...
	Importer struct {
		res             *resources.Resources
		configFile      string
		args            cli.Args
		importFiles     []string
		targetDatabase  string
		deleteOldData   bool
		userRolling     bool
		userTotalChunks int
		userCurrChunk   int
		threads         int
	}
)

//NewImporter ....
func NewImporter(c *cli.Context) *Importer {
	return &Importer{
		configFile:      getConfigFilePath(c),
		args:            c.Args(),
		deleteOldData:   c.Bool("delete"),
		userRolling:     c.Bool("rolling"),
		userTotalChunks: c.Int("numchunks"),
		userCurrChunk:   c.Int("chunk"),
		threads:         util.Max(c.Int("threads")/2, 1),
	}
}

//parseArgs handles parsing the positional import arguments
func (i *Importer) parseArgs() error {
	if len(i.args) < 2 {
		return cli.NewExitError("\n\t[!] Both <files/directory to import> and <database name> are required.", -1)
	}

	i.targetDatabase = i.args[len(i.args)-1] // the last argument
	i.importFiles = i.args[:len(i.args)-1]   // all except the last argument

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

// parseFlags validates the user supplied flags against the current state of
// the target database and determines what settings should be set in the
// rolling configuration
func parseFlags(dbExists bool, dbIsRolling bool, dbCurrChunk int, dbTotalChunks int,
	userIsRolling bool, userCurrChunk int, userTotalChunks int, cfgDefaultChunks int,
	deleteOldData bool) (config.RollingStaticCfg, error) {

	cfg := config.RollingStaticCfg{}

	// a user-provided value for either of the chunk options implies rolling
	if userTotalChunks != -1 || userCurrChunk != -1 {
		userIsRolling = true
	}

	// ensure the user specifies a rolling import if the database exists
	// and is not a rolling database.
	if !deleteOldData && (dbExists && !dbIsRolling) && !userIsRolling {
		return cfg, errors.New(
			"\t[!] New data cannot be imported into a non-rolling database. " +
				"Run with --rolling to convert this database into a rolling database",
		)
	}

	// set cfg.TotalChunks
	if userTotalChunks != -1 { // user gave the total number of chunks via command line
		// it's currently an error to try to reduce the total number of chunks in an existing rolling database
		if dbExists && dbIsRolling && userTotalChunks < dbTotalChunks {
			return cfg, fmt.Errorf(
				"\t[!] Cannot modify the total number of chunks in an existing database [ %d ]",
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
		} else if deleteOldData && dbExists && !dbIsRolling && !userIsRolling {
			// if the user is re-importing in to a non-rolling database
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

	//set cfg.CurrentChunk
	if userCurrChunk != -1 { // user gave the current chunk via command line
		// use the user-provided value
		cfg.CurrentChunk = userCurrChunk
	} else { // user didn't specify the current chunk
		if !dbExists {
			// if the database doesn't exist then assume this is the first chunk
			cfg.CurrentChunk = 0
		} else if deleteOldData && dbExists && !dbIsRolling {
			// if the user wants to re-import into a non-rolling database
			// then assume we want to replace the first chunk
			cfg.CurrentChunk = 0
		} else if deleteOldData && dbIsRolling {
			// replace the last chunk if the user specified --delete but not --chunk
			cfg.CurrentChunk = dbCurrChunk
		} else {
			// otherwise increment tne current value, wrapping back to 0 when needed
			cfg.CurrentChunk = (dbCurrChunk + 1) % cfg.TotalChunks
		}
	}

	cfg.Rolling = dbIsRolling || userIsRolling

	// preserve the default chunks setting (even though we don't use it after this currently)
	cfg.DefaultChunks = cfgDefaultChunks

	// validate current chunk number
	if cfg.CurrentChunk < 0 ||
		cfg.CurrentChunk >= cfg.TotalChunks {
		return cfg, fmt.Errorf(
			"\t[!] Current chunk number [ %d ] must be 0 or greater and less than the total number of chunks [ %d ]",
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

	i.res = resources.InitResources(i.configFile)

	// set up target database
	i.res.DB.SelectDB(i.targetDatabase)

	// set up the rolling configuration
	// grab the current rolling settings from the MetaDB
	exists, isRolling, currChunk, totalChunks, err := i.res.MetaDB.GetRollingSettings(i.targetDatabase)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("\n\t[!] Error while reading existing database settings: %v", err.Error()), -1)
	}

	// validate the user given flags against the rolling settings from the MetaDB
	// and determine the rolling configuration
	rollingCfg, err := parseFlags(
		exists, isRolling, currChunk, totalChunks,
		i.userRolling, i.userCurrChunk, i.userTotalChunks, i.res.Config.S.Rolling.DefaultChunks,
		i.deleteOldData,
	)
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}
	i.res.Config.S.Rolling = rollingCfg

	importer := parser.NewFSImporter(i.res)
	if len(importer.GetInternalSubnets()) == 0 {
		return cli.NewExitError("Internal subnets are not defined. Please set the InternalSubnets section of the config file.", -1)
	}

	indexedFiles := importer.CollectFileDetails(i.importFiles, i.threads)
	// if no compatible files for import were found, exit
	if len(indexedFiles) == 0 {
		return cli.NewExitError("No compatible log files found", -1)
	}

	if i.deleteOldData {
		err := i.handleDeleteOldData()
		if err != nil {
			return cli.NewExitError(fmt.Errorf("error deleting old data: %v", err.Error()), -1)
		}
	}

	i.res.Log.Infof("Importing %v\n", i.importFiles)
	fmt.Printf("\n\t[+] Importing %v:\n", i.importFiles)

	// about to import into and convert an existing, non-rolling database
	if exists && !isRolling && rollingCfg.Rolling {
		i.res.Log.Infof("Non-rolling database %v will be converted to rolling\n", i.targetDatabase)
		fmt.Printf("\t[+] Non-rolling database %v will be converted to rolling\n", i.targetDatabase)
	}

	/*
		// Uncomment these lines to enable CPU profiling
		f, err := os.Create("./cpu.pprof")
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	*/

	importer.Run(indexedFiles, i.threads)

	i.res.Log.Infof("Finished importing %v\n", i.importFiles)

	return nil
}

func (i *Importer) handleDeleteOldData() error {
	if !i.res.Config.S.Rolling.Rolling {
		fmt.Printf("\t[+] Removing database: %s\n", i.targetDatabase)
		err := deleteSingleDatabase(i.res, i.targetDatabase, false)
		if err != nil {
			i.res.Log.WithFields(log.Fields{
				"database": i.targetDatabase,
				"err":      err.Error(),
			}).Warn("Failed to remove database before import")

			// Don't stop execution if the old database doesn't exist.
			if err.Error() == "No records for database found" {
				fmt.Printf("\t[-] %s\n", err.Error())
			} else {
				return err
			}
		}
		return nil
	}

	// Remove the analysis results for the chunk
	targetChunk := i.res.Config.S.Rolling.CurrentChunk
	removerRepo := remover.NewMongoRemover(i.res.DB, i.res.Config, i.res.Log)
	err := removerRepo.Remove(targetChunk)
	if err != nil {
		return err
	}
	err = i.res.MetaDB.SetChunk(targetChunk, i.targetDatabase, false)
	if err != nil {
		return err
	}

	// Remove the file records so they get imported again
	err = i.res.MetaDB.RemoveFilesByChunk(i.targetDatabase, targetChunk)
	if err != nil {
		return err
	}
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
