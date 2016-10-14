package main

import (
	"os"
//	Removing this because fails when building fork
//	Remember to replace before pull request
//	"github.com/ocmdev/rita/commands"
	"github.com/bglebrun/rita/commands"

	"github.com/codegangsta/cli"
)

// Entry point of ai-hunt
func main() {
	app := cli.NewApp()
	app.Name = "rita"
	app.Usage = "Look for evil needles in big haystacks."

	// Change the version string with updates so that a quick help command will
	// let the testers know what version of HT they're on
	app.Version = "0.9 Open Beta"

	// Define commands used with this application
	app.Commands = commands.Commands()

	app.Run(os.Args)
}
