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
		Name:  "delete-database",
		Usage: "Delete an imported database",
		Flags: []cli.Flag{
			databaseFlag,
			configFlag,
		},
		Action: func(c *cli.Context) error {
			res := database.InitResources(c.String("config"))
			if c.String("database") == "" {
				return cli.NewExitError("Specify a database with -d", -1)
			}

			fmt.Println("Are you sure you want to delete database", c.String("database"), "[Y/n]")

			read := bufio.NewReader(os.Stdin)

			response, err := read.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			response = strings.ToLower(strings.TrimSpace(response))

			if response == "y" || response == "yes" {
				fmt.Println("Deleting database:", c.String("database"))
				return res.MetaDB.DeleteDB(c.String("database"))
			} else if response == "n" || response == "no" {
				return cli.NewExitError("Database "+c.String("database")+" was not deleted.", 0)
			} else {
				return cli.NewExitError("Aborted, nothing deleted.", -1)
			}
		},
	}

	bootstrapCommands(reset)
}
