package commands

import (
	"fmt"
	"time"

	"github.com/blang/semver"
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
	res := database.InitResources(configFile)
	var toRunDirty []string
	var toRun []string

	// Check to see if we want to run a full database or just one off the command line
	if inDb == "" {
		res.Log.Info("Running analysis against all databases")
		toRunDirty = append(toRun, res.MetaDB.GetUnAnalyzedDatabases()...)
	} else {
		toRunDirty = append(toRun, inDb)
	}

	// Check for problems
	for _, possDB := range toRunDirty {
		info, err := res.MetaDB.GetDBMetaInfo(possDB)
		if err != nil {
			errStr := fmt.Sprintf("Error: %s not found.", possDB)
			res.Log.Errorf(errStr)
			fmt.Println(errStr)
			continue
		}
		if info.Analyzed {
			errStr := fmt.Sprintf("Error: %s is already analyzed.", possDB)
			res.Log.Errorf(errStr)
			fmt.Println(errStr)
			continue
		}
		semVer, err := semver.ParseTolerant(info.ImportVersion)
		if err != nil {
			errStr := fmt.Sprintf("Error: %s is labelled with an incorrect version tag", possDB)
			res.Log.Errorf(errStr)
			fmt.Println(errStr)
			continue
		}
		if semVer.Major != res.Config.R.Version.Major {
			errStr := fmt.Sprintf("Error: %s was parsed by an incompatible version of RITA", possDB)
			res.Log.Errorf(errStr)
			fmt.Println(errStr)
			continue
		}
		toRun = append(toRun, possDB)
	}

	startAll := time.Now()

	fmt.Println("[+] Analyzing:")
	for _, db := range toRun {
		fmt.Println("\t[-] " + db)
	}
	res.Log.WithFields(log.Fields{
		"databases":  toRun,
		"start_time": startAll.Format(util.TimeFormat),
	}).Info("Preparing to analyze ")

	for _, td := range toRun {
		startIndiv := time.Now()
		res.Log.WithFields(log.Fields{
			"database":   td,
			"start_time": startIndiv.Format(util.TimeFormat),
		}).Info("Analyzing")
		fmt.Println("[+] Analyzing " + td)
		res.DB.SelectDB(td)

		sanitization.SanitizeData(res)

		logAnalysisFunc("Unique Connections", td, res,
			structure.BuildUniqueConnectionsCollection,
		)
		logAnalysisFunc("Unique Hosts", td, res,
			func(innerRes *database.Resources) {
				structure.BuildHostsCollection(innerRes)
				structure.BuildIPv4Collection(innerRes)
				structure.BuildIPv6Collection(innerRes)
			},
		)
		logAnalysisFunc("Unique Hostnames", td, res,
			dns.BuildHostnamesCollection,
		)
		logAnalysisFunc("Exploded DNS", td, res,
			dns.BuildExplodedDNSCollection,
		)
		logAnalysisFunc("URL Length", td, res,
			urls.BuildUrlsCollection,
		)
		logAnalysisFunc("User Agent", td, res,
			useragent.BuildUserAgentCollection,
		)
		logAnalysisFunc("Blacklisted", td, res,
			blacklist.BuildBlacklistedCollections,
		)
		logAnalysisFunc("Beaconing", td, res,
			beacon.BuildBeaconCollection,
		)
		logAnalysisFunc("Scanning", td, res,
			scanning.BuildScanningCollection,
		)
		logAnalysisFunc("Cross Reference", td, res,
			crossref.BuildXRefCollection,
		)

		res.MetaDB.MarkDBAnalyzed(td, true)
		endIndiv := time.Now()
		res.Log.WithFields(log.Fields{
			"database": td,
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
	resources *database.Resources, analysis func(*database.Resources)) {
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
