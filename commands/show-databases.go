package commands

import (
	"fmt"

	"github.com/bglebrun/rita/database"
	"github.com/urfave/cli"
)

func init() {

	databases := cli.Command{
		Name:  "show-databases",
		Usage: "Print the databases currently stored",
		Action: func(c *cli.Context) error {
			res := database.InitResources("")
			for _, name := range res.MetaDB.GetDatabases() {
				fmt.Println(name)
			}
			return nil
		},
	}

	bootstrapCommands(databases)
}
