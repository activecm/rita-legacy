package commands

import (
	"fmt"

	"github.com/ocmdev/rita/database"
	"github.com/urfave/cli"
)

func init() {

	databases := cli.Command{
		Name:  "show-databases",
		Usage: "Print the databases currently stored",
		Flags: []cli.Flag{
			configFlag,
		},
		Action: func(c *cli.Context) error {
			res := database.InitResources(c.String("config"))
			for _, name := range res.MetaDB.GetDatabases() {
				fmt.Println(name)
			}
			return nil
		},
	}

	bootstrapCommands(databases)
}
