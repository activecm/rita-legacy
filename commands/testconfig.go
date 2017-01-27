package commands

import (
	"fmt"
	"os"

	"github.com/ocmdev/rita/config"
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
		Name:   "testconfig",
		Usage:  "Check the configuration file for validity",
		Action: testConfiguration,
	}

	allCommands = append(allCommands, command)
}

// testConfiguration prints out the result of parsing the config file
func testConfiguration(c *cli.Context) error {
	cfg := config.InitConfig(c.String("config"))

	yml, err := yaml.Marshal(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(os.Stdout, "\n%s\n", string(yml))
	return nil
}
