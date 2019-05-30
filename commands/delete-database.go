package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/urfave/cli"
)

func init() {
	reset := cli.Command{
		Name:      "delete",
		Aliases:   []string{"delete-database"},
		Usage:     "Delete an imported database",
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

	// get all database names
	names, err := res.DB.Session.DatabaseNames()

	// check if we have databases
	if err != nil || len(names) == 0 {
		return cli.NewExitError("Failed to find any databases", -1)
	}

	// For single database deletion
	if !c.Bool("all") {

		database := c.Args().Get(0)
		if database == "" {
			return cli.NewExitError("Specify a database or specify --all or -a for all databases", -1)
		}

		// if no force flag, verify action with the user
		if !c.Bool("force") {
			if confirmAction("Are you sure you want to delete database " + database + " [y/N] ") {
				fmt.Println("Deleting database: ", database)
			} else {
				return cli.NewExitError("Database "+database+" was not deleted.", 0)
			}
		}

		dberr := deleteSingleDatabase(res, names, database, c.Bool("dry-run"))
		if dberr != nil {
			return cli.NewExitError(dberr.Error, -1)
		}

	} else {
		// Otherwise, we're deleting everything
		if !c.Bool("force") {
			if confirmAction("Confirm we'll be deleting the following databases:\n" + strings.Join(names, "\n")) {
				fmt.Println("Deleting everything...")
			} else {
				return cli.NewExitError("Nothing deleted, no changes have been made", 0)
			}
		}

		for _, database := range names {
			dberr := deleteSingleDatabase(res, names, database, c.Bool("dry-run"))
			if dberr != nil {
				return cli.NewExitError(dberr.Error, -1)
			}
		}

	}

	if c.Bool("dry-run") {
		cli.NewExitError("This was a dry run of the delete command, nothing has been changed!", 0)
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
	fmt.Print(confimationMessage)

	read := bufio.NewReader(os.Stdin)

	response, _ := read.ReadString('\n')

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "y" || response == "yes" {
		return true
	}

	return false
}
