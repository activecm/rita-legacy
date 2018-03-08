package main

import (
	"os"
	"runtime"

	"github.com/activecm/rita/commands"
	"github.com/activecm/rita/config"
	"github.com/urfave/cli"
)

// Entry point of ai-hunt
func main() {
	app := cli.NewApp()
	app.Name = "rita"
	app.Usage = "Look for evil needles in big haystacks."

	// Change the version string with updates so that a quick help command will
	// let the testers know what version of HT they're on
	app.Version = config.Version

	// Define commands used with this application
	app.Commands = commands.Commands()

	runtime.GOMAXPROCS(runtime.NumCPU())
	app.Run(os.Args)
}
