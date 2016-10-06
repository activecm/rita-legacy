package commands

import (
	"fmt"
	"io/ioutil"
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

// checkConfig checks to see if the config file provided unmarshals cleanly
func checkConfig(path string) {
	var cfg config.SystemConfig
	cfFile, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("testconfig: Error reading config file: %s\n", err)
		return
	}
	err = yaml.Unmarshal(cfFile, &cfg)
	if err != nil {
		fmt.Printf("testconfig: Error unmarshalling file: \n%s\n", err)
		return
	}

	ymlConf, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Println("There was an error unmarshalling the yaml file,", err)
		return
	}
	fmt.Printf("\n%s\n", string(ymlConf))
	fmt.Printf("No errors found.\n")
	return
}
