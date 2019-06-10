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
			forceFlag,
			configFlag,
			allFlag,
			matchFlag,
			regexFlag,
			dryRunFlag,
		},
		Action: deleteDatabase,
	}

	bootstrapCommands(reset)
}

//deleteDatabase deletes a target database
func deleteDatabase(c *cli.Context) error {
	res := resources.InitResources(c.String("config"))

	// Different command flags
	match := c.String("match")
	regex := c.String("regex")
	bulk := c.Bool("all")
	force := c.Bool("force")
	dryRun := c.Bool("dry-run")
	var names []string

	if checkFlags((match != ""), (regex != ""), bulk) {
		return cli.NewExitError("Please select a single bulk option", -1)
	}

	if bulk {
		dbs := res.MetaDB.GetDatabases()
		copy(names, dbs)
	} else if match != "" {

		// Get DB list
		dbs := res.MetaDB.GetDatabases()

		// Find dbs with matching names
		for _, db := range dbs {
			if strings.Contains(db, match) {
				names = append(names, db)
			}
		}

	} else if regex != "" {

		// Get DB list
		dbs := res.MetaDB.GetDatabases()

		// Compile regex, check if it's valid
		regq, err := regexp.Compile(regex)
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
		// Single database deletion
		tgtDB := c.Args().Get(0)
		// get all database names
		dbs := res.MetaDB.GetDatabases()
		if find(dbs, tgtDB) != len(dbs) {
			names = append(names, tgtDB)
		}

	}

	// check if we have databases
	if len(names) == 0 {
		return cli.NewExitError("Failed to find any databases", -1)
	}

	// if no force or dry run flag, verify action
	if !force && !dryRun {
		if confirmAction("Confirm we'll be deleting the following databases:\n" + strings.Join(names, "\n")) {
			fmt.Println("Deleting databases...")
		} else {
			return cli.NewExitError("Nothing deleted, no changes have been made", 0)
		}
	}

	// Iterate through databases to delete and delete them one by one
	for _, database := range names {
		dberr := deleteSingleDatabase(res, names, database, dryRun)
		if dberr != nil {
			return cli.NewExitError(dberr.Error, -1)
		}
	}

	// Dry run warning
	if dryRun {
		fmt.Fprintf(os.Stdout, "\t[-] This was a dry run of the delete command, nothing has been changed!\n")
	}

	return nil
}

func deleteSingleDatabase(res *resources.Resources, dbnames []string, db string, dryRun bool) error {
	// check if database exists
	dbExists := util.StringInSlice(db, dbnames)

	// check if metadatabase record for database exists
	mDBExists := util.StringInSlice(db, res.MetaDB.GetDatabases())

	if !dryRun {
		// delete database if it exists
		if dbExists {
			if res.DB.Session.DB(db).DropDatabase() != nil {
				return errors.New("Failed to delete database")
			}
		}

		// remove metadatabase record if it exists
		if mDBExists {
			if res.MetaDB.DeleteDB(db) != nil {
				return errors.New("Failed to update metadb")
			}
		}
	}

	// check if we have anything to delete
	if !dbExists && !mDBExists {
		return errors.New("No records for database found")
	}

	// if it got here, deleting was a success!
	fmt.Fprintf(os.Stdout, "\t[-] Successfully deleted database %s.\n", db)

	return nil
}

func confirmAction(confimationMessage string) bool {
	fmt.Print(confimationMessage, "\n [y/N] : ")

	read := bufio.NewReader(os.Stdin)

	response, _ := read.ReadString('\n')

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "y" || response == "yes" {
		return true
	}

	return false
}

func checkFlags(a bool, b bool, c bool) bool {
	return xor3(a, b, c) && !(a && b && c)
}

// Go for some reason doesn't have a boolean xor operator
// which I insanely love, thank you google
func xor3(a bool, b bool, c bool) bool {
	return (!a && b && !c) || (a && !b && !c) || (!a && !b && c) || (a && b && c)
}

func find(array []string, search string) int {
	for i, n := range array {
		if search == n {
			return i
		}
	}
	return len(search)
}
