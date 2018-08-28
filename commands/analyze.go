package commands

import (
	"fmt"
	"time"

	"github.com/activecm/rita/analysis/beacon"
	"github.com/activecm/rita/analysis/blacklist"
	"github.com/activecm/rita/analysis/crossref"
	"github.com/activecm/rita/analysis/dns"
	"github.com/activecm/rita/analysis/sanitization"
	"github.com/activecm/rita/analysis/scanning"
	"github.com/activecm/rita/analysis/structure"
	"github.com/activecm/rita/analysis/urls"
	"github.com/activecm/rita/analysis/useragent"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func init() {
	analyzeCommand := cli.Command{
		Name:  "analyze",
		Usage: "Analyze imported databases",
		UsageText: "rita analyze [command options] [database]\n\n" +
			"If no database is specified, every database will be analyzed.",
		Flags: []cli.Flag{
			configFlag,
		},
		Action: func(c *cli.Context) error {
			return analyze(c.Args().Get(0), c.String("config"))
		},
	}

	bootstrapCommands(analyzeCommand)
}

func analyze(inDb string, configFile string) error {
	res := resources.InitResources(configFile)
	var toRunDirty []database.RITADatabase
	var toRun []database.RITADatabase

	// Check to see if we want to run a full database or just one off the command line
	if inDb == "" {
		res.Log.Info("Running analysis against all unanalyzed databases")
		unanalyzedDBs, err := res.DBIndex.GetUnanalyzedDatabases()
		if err != nil {
			errStr := fmt.Sprintf("Error: failed to list unanalyzed databases: %s", err.Error())
			res.Log.Error(errStr)
			return cli.NewExitError(errStr, 255)
		}

		//ensure the databases have finished being imported
		for i := range unanalyzedDBs {
			if unanalyzedDBs[i].ImportFinished() {
				toRunDirty = append(toRunDirty, unanalyzedDBs[i])
			}
		}
	} else {
		specifiedDB, err := res.DBIndex.GetDatabase(inDb)
		if err != nil {
			errStr := fmt.Sprintf("Error: database %s not found", inDb)
			res.Log.Error(errStr)
			return cli.NewExitError(errStr, 255)
		}
		if specifiedDB.Analyzed() {
			errStr := fmt.Sprintf("Error: %s is already analyzed.", specifiedDB.Name())
			res.Log.Error(errStr)
			return cli.NewExitError(errStr, 255)
		}
		if !specifiedDB.ImportFinished() {
			errStr := fmt.Sprintf("Error: %s hasn't finished importing.", specifiedDB.Name())
			res.Log.Error(errStr)
			return cli.NewExitError(errStr, 255)
		}
		toRunDirty = append(toRunDirty, specifiedDB)
	}

	// Check for version problems
	for i := range toRunDirty {
		compatible, err := toRunDirty[i].CompatibleImportVersion(res.Config.R.Version)
		if err != nil {
			errStr := fmt.Sprintf("Error: %s is labelled with an incorrect version tag", toRunDirty[i].Name())
			res.Log.Error(errStr)
			fmt.Println(errStr)
			continue
		}
		if !compatible {
			errStr := fmt.Sprintf("Error: %s was imported with an incompatible version of RITA", toRunDirty[i].Name())
			res.Log.Error(errStr)
			fmt.Println(errStr)
			continue
		}
		toRun = append(toRun, toRunDirty[i])
	}

	//If theres no databases to analyze after filtering everything out
	//exit
	if len(toRun) == 0 {
		return cli.NewExitError("", 255)
	}

	startAll := time.Now()

	fmt.Println("[+] Analyzing:")
	for i := range toRun {
		fmt.Println("\t[-] " + toRun[i].Name())
	}

	var dbNames []string
	for i := range toRun {
		dbNames = append(dbNames, toRun[i].Name())
	}
	res.Log.WithFields(log.Fields{
		"databases":  dbNames,
		"start_time": startAll.Format(util.TimeFormat),
	}).Info("Preparing to analyze ")

	for _, ritaDB := range toRun {
		startIndiv := time.Now()
		res.Log.WithFields(log.Fields{
			"database":   ritaDB.Name(),
			"start_time": startIndiv.Format(util.TimeFormat),
		}).Info("Analyzing")
		fmt.Println("[+] Analyzing " + ritaDB.Name())
		res.DB.SelectDB(ritaDB.Name())

		sanitization.SanitizeData(res)

		logAnalysisFunc("Unique Connections", ritaDB.Name(), res,
			structure.BuildUniqueConnectionsCollection,
		)
		logAnalysisFunc("Unique Hosts", ritaDB.Name(), res,
			func(innerRes *resources.Resources) {
				structure.BuildHostsCollection(innerRes)
				structure.BuildIPv4Collection(innerRes)
				structure.BuildIPv6Collection(innerRes)
			},
		)
		logAnalysisFunc("Unique Hostnames", ritaDB.Name(), res,
			dns.BuildHostnamesCollection,
		)
		logAnalysisFunc("Exploded DNS", ritaDB.Name(), res,
			dns.BuildExplodedDNSCollection,
		)
		logAnalysisFunc("URL Length", ritaDB.Name(), res,
			urls.BuildUrlsCollection,
		)
		logAnalysisFunc("User Agent", ritaDB.Name(), res,
			useragent.BuildUserAgentCollection,
		)
		logAnalysisFunc("Blacklisted", ritaDB.Name(), res,
			blacklist.BuildBlacklistedCollections,
		)
		logAnalysisFunc("Beaconing", ritaDB.Name(), res,
			beacon.BuildBeaconCollection,
		)
		logAnalysisFunc("Scanning", ritaDB.Name(), res,
			scanning.BuildScanningCollection,
		)
		logAnalysisFunc("Cross Reference", ritaDB.Name(), res,
			crossref.BuildXRefCollection,
		)

		err := ritaDB.SetAnalyzed(res.DB.Session, res.Config.R.Version)
		if err != nil {
			errStr := fmt.Sprintf("Error: could not mark %s as analyzed: %s", ritaDB.Name(), err.Error())
			res.Log.Error(errStr)
			fmt.Println(errStr)
			return cli.NewExitError(errStr, 255)
		}

		endIndiv := time.Now()
		res.Log.WithFields(log.Fields{
			"database": ritaDB.Name(),
			"end_time": endIndiv.Format(util.TimeFormat),
			"duration": endIndiv.Sub(startIndiv),
		}).Info("Analysis complete")
	}
	endAll := time.Now()
	res.Log.WithFields(log.Fields{
		"end_time": endAll.Format(util.TimeFormat),
		"duration": endAll.Sub(startAll),
	}).Info("Analysis complete")
	return nil
}

func logAnalysisFunc(analysisName string, databaseName string,
	resources *resources.Resources, analysis func(*resources.Resources)) {
	analysisName += " Analysis"
	start := time.Now()
	resources.Log.WithFields(log.Fields{
		"analysis":   analysisName,
		"database":   databaseName,
		"start_time": start.Format(util.TimeFormat),
	}).Infof("Running analysis")
	fmt.Println("\t[-] Running " + analysisName)
	analysis(resources)
	end := time.Now()
	resources.Log.WithFields(log.Fields{
		"analysis": analysisName,
		"database": databaseName,
		"end_time": end.Format(util.TimeFormat),
		"duration": end.Sub(start),
	}).Infof("Analysis complete")
}
