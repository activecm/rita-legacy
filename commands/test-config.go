package commands

import (
	"fmt"
	"os"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/resources"

	"github.com/urfave/cli"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	command := cli.Command{
		Flags:  []cli.Flag{ConfigFlag},
		Name:   "test-config",
		Usage:  "Check the configuration file for validity",
		Before: SetConfigFilePath,
		Action: testConfiguration,
	}

	allCommands = append(allCommands, command)
}

// testConfiguration prints out the result of parsing the config file
func testConfiguration(c *cli.Context) error {
	// First, print out the config as it was parsed
	conf, err := config.LoadConfig(getConfigFilePath(c))
	if err != nil {
		fmt.Fprintf(os.Stdout, "Failed to config: %s\n", err.Error())
		os.Exit(-1)
	}

	staticConfig, err := yaml.Marshal(conf.S)
	if err != nil {
		return err
	}

	tableConfig, err := yaml.Marshal(conf.T)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "\n%s\n", string(staticConfig))
	fmt.Fprintf(os.Stdout, "\n%s\n", string(tableConfig))

	// Then test initializing external resources like db connection and file handles
	resources.InitResources(getConfigFilePath(c))

	return nil
}
