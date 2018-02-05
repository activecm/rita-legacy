package commands

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ocmdev/rita/database"
	"github.com/urfave/cli"
)

func init() {
	reset := cli.Command{
		Name:      "delete-database",
		Usage:     "Delete an imported database",
		ArgsUsage: "<database>",
		Flags: []cli.Flag{
			configFlag,
		},
		Action: func(c *cli.Context) error {
			res := database.InitResources(c.String("config"))
			db := c.Args().Get(0)
			if db == "" {
				return cli.NewExitError("Specify a database", -1)
			}

			fmt.Print("Are you sure you want to delete database ", db, " [y/N] ")

			read := bufio.NewReader(os.Stdin)

			response, err := read.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			response = strings.ToLower(strings.TrimSpace(response))

			if response == "y" || response == "yes" {
				fmt.Println("Deleting database:", db)
				err = res.MetaDB.DeleteDB(db)
				if err != nil {
					return cli.NewExitError("ERROR: "+err.Error(), -1)
				}
			} else {
				return cli.NewExitError("Database "+db+" was not deleted.", 0)
			}
			return nil
		},
	}

	bootstrapCommands(reset)
}
