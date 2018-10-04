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
			dbs, err := res.DBIndex.GetDatabases()
			if err != nil {
				return cli.NewExitError("Error: could not list databases: "+err.Error(), -1)
			}
			for i := range dbs {
				fmt.Println(dbs[i].Name())
			}
			return nil
		},
	}

	bootstrapCommands(databases)
}
