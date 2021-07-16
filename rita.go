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
	app.Flags = []cli.Flag{commands.ConfigFlag}

	cli.VersionPrinter = commands.GetVersionPrinter()

	// Change the version string with updates so that a quick help command will
	// let the testers know what version of HT they're on
	app.Version = config.Version
	app.EnableBashCompletion = true

	// Define commands used with this application
	app.Commands = commands.Commands()
	app.Before = commands.SetConfigFilePath

	runtime.GOMAXPROCS(runtime.NumCPU())
	app.Run(os.Args)
}
