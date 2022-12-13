package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/urfave/cli"
)

func init() {
	reset := cli.Command{
		Name:      "delete",
		Aliases:   []string{"delete-database"},
		Usage:     "Delete imported database(s)",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			ConfigFlag,
			forceFlag,
			allFlag,
			matchFlag,
			regexFlag,
			dryRunFlag,
		},
		Action: deleteDatabase,
	}

	bootstrapCommands(reset)
}

// deleteDatabase deletes a target database
func deleteDatabase(c *cli.Context) error {
	res := resources.InitResources(getConfigFilePath(c))

	// Different command flags
	tgt := c.Args().Get(0)
	match := c.Bool("match")
	regex := c.Bool("regex")
	bulk := c.Bool("all")
	force := c.Bool("force")
	dryRun := c.Bool("dry-run")
	var names []string

	err := checkCommandFlags(match, regex, bulk, tgt)
	if err != nil {
		return cli.NewExitError(err.Error(), -1)
	}

	if match {
		// Get DB list
		dbs := res.MetaDB.GetDatabases()

		// Find dbs with matching names
		for _, db := range dbs {
			if strings.Contains(db, tgt) {
				names = append(names, db)
			}
		}

	} else if regex {
		// Get DB list
		dbs := res.MetaDB.GetDatabases()

		// Compile regex, check if it's valid
		regq, err := regexp.Compile(tgt)
		if err != nil {
			return cli.NewExitError(err.Error, -1)
		}

		// Find dbs with regex query
		for _, db := range dbs {
			matched := regq.MatchString(db)
			if matched {
				names = append(names, db)
			}
		}

	} else {
		// get all database names
		dbs := res.MetaDB.GetDatabases()
		//if all database is selected we append all dbs to be deleted, otherwise
		//  check if we are getting other dbs
		if bulk {
			names = append(names, dbs...)
		} else {
			if util.StringInSlice(tgt, dbs) {
				names = append(names, tgt)
			}
		}
	}

	// check if we have databases
	if len(names) == 0 {
		return cli.NewExitError("Failed to find any databases", -1)
	}

	// if no force or dry run flag, verify action
	if !force && !dryRun {
		if confirmAction("Confirm we'll be deleting the following databases:\n" + strings.Join(names, "\n") + "\n") {
			fmt.Println("Deleting databases...")
		} else {
			return cli.NewExitError("Nothing deleted, no changes have been made", 0)
		}
	}

	// Iterate through databases to delete and delete them one by one
	for _, database := range names {
		dberr := deleteSingleDatabase(res, database, dryRun)
		if dberr != nil {
			return cli.NewExitError(dberr.Error, -1)
		}
	}

	// Dry run warning
	if dryRun {
		fmt.Printf("\t[-] This was a dry run of the delete command, nothing has been changed!\n")
	}

	return nil
}

func deleteSingleDatabase(res *resources.Resources, db string, dryRun bool) error {
	// check if database exists
	collNames, err := res.DB.Session.DB(db).CollectionNames()
	if err != nil {
		return err
	}
	dbExists := len(collNames) != 0
	// check if metadatabase record for database exists
	mDBExists := util.StringInSlice(db, res.MetaDB.GetDatabases())

	if !dryRun {
		// delete database if it exists
		if dbExists {
			if res.DB.Session.DB(db).DropDatabase() != nil {
				return errors.New("failed to delete database")
			}
		}

		// remove metadatabase record if it exists
		if mDBExists {
			if res.MetaDB.DeleteDB(db) != nil {
				return errors.New("failed to update metadb")
			}
		}
	}

	// check if we have anything to delete
	if !dbExists && !mDBExists {
		return errors.New("no records for database found")
	}

	// if it got here, deleting was a success!
	fmt.Printf("\t[-] Successfully deleted database %s.\n", db)

	return nil
}

// Confirms action, takes a string that is the confirmation message,
// returns true if the user has selected true, and false
// if the user answers otherwise (assumed no)
func confirmAction(confimationMessage string) bool {
	fmt.Print(confimationMessage + " [y/N]: ")

	read := bufio.NewReader(os.Stdin)

	response, _ := read.ReadString('\n')

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "y" || response == "yes" {
		return true
	}

	return false
}

func checkCommandFlags(match, regex, bulk bool, tgt string) error {
	// All fields empty, if we have the all flag set, don't need a database name
	if tgt == "" && !bulk {
		return errors.New("please provide a database or string parameter or invoke with `--help` or `-h` for usage")
	}

	// Flags
	if !checkFlagsExclusive(bulk, match, regex) {
		return errors.New("invalid combination of flags, invoke with `--help` or `-h` for usage")
	}

	return nil
}

// Checks if 3 bool flags are exclusively set,
// If only a single flag is set, returns true, otherwise
// returns false if more than a single flag is set
// also allows a single database to be deleted if no flag is set
func checkFlagsExclusive(a, b, c bool) bool {
	return (!a && b && !c) || (a && !b && !c) || (!a && !b && c || (!a && !b && !c))
}
