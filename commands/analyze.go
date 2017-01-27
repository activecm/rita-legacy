package commands

import (
	"fmt"
	"os"

	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/intel"
	"github.com/urfave/cli"
)

func init() {
	analyzeCommand := cli.Command{
		Name:  "analyze",
		Usage: "Analyze imported databases, if no [database,d] flag is specified will attempt all",
		Flags: []cli.Flag{
			databaseFlag,
		},
		Action: func(c *cli.Context) error {
			analyze(c.String("database"))
			return nil
		},
	}

	bootstrapCommands(analyzeCommand)
}

func analyze(inDb string) {

	conf := config.InitConfig("")
	dbm := database.NewMetaDBHandle(conf)
	var toRun []string

	// Check to see if we want to run a full database or just one off the command line
	if inDb == "" {
		fmt.Println("Running analysis against all databases")
		names := dbm.GetUnAnalyzedDatabases()
		fmt.Println("Preparing to analyze these databases:")
		for _, db := range names {
			fmt.Println(db)
			toRun = append(toRun, db)
		}
	} else {
		info, err := dbm.GetDBMetaInfo(inDb)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error: %s not found.\n", inDb)
			return
		}
		if info.Analyzed {
			fmt.Fprintf(os.Stdout, "Error: %s is already analyzed.\n", inDb)
			return
		}

		fmt.Fprintf(os.Stdout,
			"Preparing to analyze %s\n",
			inDb)
		toRun = append(toRun, inDb)
	}

	// TODO: Thread this
	for _, td := range toRun {
		fmt.Fprintf(os.Stdout,
			"[!] Constructing databases for %s\n", td)

		conf.System.DB = td
		d := database.NewDB(conf)

		fmt.Fprintf(os.Stdout,
			"\t[-] Building the Unique Connections Collection\n")

		d.BuildUniqueConnectionsCollection()

		fmt.Fprintf(os.Stdout,
			"\t[-] Building the Hosts Collection\n")
		d.BuildHostsCollection()

		fmt.Fprintf(os.Stdout,
			"\t[-] Building the URL collection\n")

		d.BuildUrlsCollection()

		// The intel module leans on the hostnames collection
		fmt.Fprintf(os.Stdout,
			"\t[-] Building the hostnames collection\n")

		d.BuildHostnamesCollection()

		fmt.Fprintf(os.Stdout,
			"\t[-] Adding new external hosts to the intelligence database\n")

		itl := intel.NewIntelHandle(conf)
		itl.Run()

		// Module Collections
		fmt.Fprintf(os.Stdout,
			"\t[-] Running beacon analysis\n")

		d.BuildTBDCollection()

		fmt.Fprintf(os.Stdout,
			"\t[-] Running blacklisted analysis\n")

		d.BuildBlacklistedCollection()

		fmt.Fprintf(os.Stdout,
			"\t[-] Running useragent analysis\n")

		d.BuildUserAgentCollection()

		fmt.Fprintf(os.Stdout,
			"\t[-] Searching for port scanning activity\n")
		d.BuildScanningCollection()

		dbm.MarkDBCompleted(td, true)
		fmt.Fprintf(os.Stdout,
			"[+] %s analysis complete!\n", td)
	}
}
