package commands

import (
	"fmt"
	"os"

	"github.com/activecm/rita/database"
	"github.com/urfave/cli"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	command := cli.Command{
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config, c",
				Usage: "specify a config file to be used",
				Value: "",
			},
		},
		Name:   "test-config",
		Usage:  "Check the configuration file for validity",
		Action: testConfiguration,
	}

	allCommands = append(allCommands, command)
}

// testConfiguration prints out the result of parsing the config file
func testConfiguration(c *cli.Context) error {
	res := database.InitResources(c.String("config"))

	staticConfig, err := yaml.Marshal(res.Config.S)
	if err != nil {
		return err
	}

	tableConfig, err := yaml.Marshal(res.Config.T)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "\n%s\n", string(staticConfig))
	fmt.Fprintf(os.Stdout, "\n%s\n", string(tableConfig))

	return nil
}
