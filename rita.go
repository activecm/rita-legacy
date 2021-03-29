package main

import (
	"os"
	"runtime"

	"github.com/activecm/rita/commands"
	"github.com/activecm/rita/config"
	"github.com/urfave/cli"
)

// Entry point of ac-hunt
func main() {
	app := cli.NewApp()
	app.Name = "rita"
	app.Usage = "Look for evil needles in big haystacks."

	// configFlag allows users to specify an alternate config file to use
	configFlag := cli.StringFlag{
		Name:  "config, c",
		Usage: "Use a given `CONFIG_FILE` when running this command",
		Value: "",
	}
	app.Flags = []cli.Flag{configFlag}

	cli.VersionPrinter = commands.GetVersionPrinter()

	// Change the version string with updates so that a quick help command will
	// let the testers know what version of HT they're on
	app.Version = config.Version
	app.EnableBashCompletion = true

	// Define commands used with this application
	app.Commands = commands.Commands()

	runtime.GOMAXPROCS(runtime.NumCPU())
	app.Run(os.Args)
}
