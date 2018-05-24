package commands

import (
	"fmt"

	"github.com/activecm/rita/resources"
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
			res := resources.InitResources(c.String("config"))
			for _, name := range res.MetaDB.GetDatabases() {
				fmt.Println(name)
			}
			return nil
		},
	}

	bootstrapCommands(databases)
}
