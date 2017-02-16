package commands

import (
	"fmt"
	"os"

	"github.com/ocmdev/rita/analysis/TBD"
	"github.com/ocmdev/rita/analysis/blacklisted"
	"github.com/ocmdev/rita/analysis/crossref"
	"github.com/ocmdev/rita/analysis/scanning"
	"github.com/ocmdev/rita/analysis/structure"
	"github.com/ocmdev/rita/analysis/urls"
	"github.com/ocmdev/rita/analysis/useragent"
	"github.com/ocmdev/rita/database"
	"github.com/urfave/cli"
)

func init() {
	analyzeCommand := cli.Command{
		Name:  "analyze",
		Usage: "Analyze imported databases, if no [database,d] flag is specified will attempt all",
		Flags: []cli.Flag{
			databaseFlag,
			configFlag,
		},
		Action: func(c *cli.Context) error {
			analyze(c.String("database"), c.String("config"))
			return nil
		},
	}

	bootstrapCommands(analyzeCommand)
}

func analyze(inDb string, configFile string) {
	res := database.InitResources(configFile)
	var toRun []string

	// Check to see if we want to run a full database or just one off the command line
	if inDb == "" {
		fmt.Println("Running analysis against all databases")
		names := res.MetaDB.GetUnAnalyzedDatabases()
		fmt.Println("Preparing to analyze these databases:")
		for _, db := range names {
			fmt.Println(db)
			toRun = append(toRun, db)
		}
	} else {
		info, err := res.MetaDB.GetDBMetaInfo(inDb)
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

	for _, td := range toRun {
		fmt.Fprintf(os.Stdout,
			"[!] Constructing databases for %s\n", td)

		res.DB.SelectDB(td)

		fmt.Fprintf(os.Stdout,
			"\t[-] Building the Unique Connections Collection\n")

		structure.BuildUniqueConnectionsCollection(res)

		fmt.Fprintf(os.Stdout,
			"\t[-] Building the Hosts Collection\n")
		structure.BuildHostsCollection(res)

		fmt.Fprintf(os.Stdout,
			"\t[-] Building the URL collection\n")

		urls.BuildUrlsCollection(res)

		fmt.Fprintf(os.Stdout,
			"\t[-] Building the hostnames collection\n")

		urls.BuildHostnamesCollection(res)

		// Module Collections
		fmt.Fprintf(os.Stdout,
			"\t[-] Running beacon analysis\n")

		TBD.BuildTBDCollection(res)

		fmt.Fprintf(os.Stdout,
			"\t[-] Running blacklisted analysis\n")

		blacklisted.BuildBlacklistedCollection(res)

		fmt.Fprintf(os.Stdout,
			"\t[-] Running useragent analysis\n")

		useragent.BuildUserAgentCollection(res)

		fmt.Fprintf(os.Stdout,
			"\t[-] Searching for port scanning activity\n")
		scanning.BuildScanningCollection(res)

		fmt.Fprintf(os.Stdout,
			"\t[-] Running cross-reference analysis\n")
		crossref.BuildXRefCollection(res)

		res.MetaDB.MarkDBAnalyzed(td, true)
		fmt.Fprintf(os.Stdout,
			"[+] %s analysis complete!\n", td)
	}
}
