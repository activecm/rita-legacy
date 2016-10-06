package commands

import (
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/ocmdev/rita/datatypes/scanning"

	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/intel"

	log "github.com/Sirupsen/logrus"
	"github.com/ocmdev/rita/datatypes/TBD"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func analyze(inDb string, verboseFlag bool) {
	start := time.Now()
	conf := config.InitConfig("")
	var torun []string

	// Check to see if we want to run a full database or just one off the command line
	if inDb == "" {
		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"Running analysis against %s \n",
				conf.System.BroConfig.MetaDB)
		}

		ssn := conf.Session.Copy()
		defer ssn.Close()
		iter := ssn.DB(conf.System.BroConfig.MetaDB).
			C("databases").Find(nil).Iter()
		var tdb database.DBMetaInfo

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"Preparing to analyze these databases:\n")
		}

		for iter.Next(&tdb) {
			if tdb.Analysed {
				continue
			}

			if verboseFlag {
				fmt.Fprintf(os.Stdout, "%s \n", tdb.Name)
			}

			torun = append(torun, tdb.Name)
		}
	} else {

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"Preparing to analyze %s\n",
				inDb)
		}
		torun = append(torun, inDb)
	}

	dbm := database.NewMetaDBHandle(conf)

	// TODO: Thread this
	for _, td := range torun {

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"[!] Constructing databases for %s\n", td)
		}
		conf.System.DB = td
		d := database.NewDB(conf)

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Building the Unique Connections Collection\n")
		}

		d.BuildUniqueConnectionsCollection()

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Building the Hosts Collection\n")
		}

		d.BuildHostsCollection()

		// The intel module leans on the hostnames collection

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Building the URL collection\n")
		}

		d.BuildUrlsCollection()

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Building the hostnames collection\n")
		}

		d.BuildHostnamesCollection()

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Adding new external hosts to the intelligence database\n")
		}

		itl := intel.NewIntelHandle(conf)
		itl.Run()

		// Server Collections

		// Module Collections

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Running beacon analysis\n")
		}

		d.BuildTBDCollection()

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Running blacklisted analysis\n")
		}

		d.BuildBlacklistedCollection()

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Running useragent analysis\n")
		}

		d.BuildUserAgentCollection()

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"\t[-] Searching for port scanning activity\n")
		}

		d.BuildScanningCollection()

		dbm.MarkDBCompleted(td)

		if verboseFlag {
			fmt.Fprintf(os.Stdout,
				"[+] %s analysis complete!\n", td)
		}

	}
	conf.Log.WithFields(log.Fields{
		"time_elapsed": time.Since(start).Seconds(),
	}).Info("completed analysis")

}

// cleanAnalysis cleans out all of the analysis data, leaving behind only the
// raw data from parsing the logs
func cleanAnalysis(dataset string) {
	conf := config.InitConfig("")
	conf.System.DB = dataset

	conn := conf.System.StructureConfig.ConnTable
	http := conf.System.StructureConfig.HttpTable
	dns := conf.System.DnsConfig.DnsTable
	names, err := conf.Session.DB(dataset).CollectionNames()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get collection names: %s\n", err.Error())
		os.Exit(-1)
	}

	for _, name := range names {
		switch name {
		case conn, http, dns:
			continue
		default:
			err := conf.Session.DB(dataset).C(name).DropCollection()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to drop collection: %s\n", err.Error())
			}
		}
	}

}

// cleanAnalysisAll uses the metadb to walk all databases and clean the analysis
func cleanAnalysisAll(dataset string) {
	conf := config.InitConfig("")
	conf.System.DB = dataset

	coll := conf.Session.DB(dataset).C("databases")
	iter := coll.Find(nil).Iter()

	var dbinfo struct {
		ID       bson.ObjectId `bson:"_id"`
		Name     string        `bson:"name"`
		Analyzed bool          `bson:"analyzed"`
	}

	for iter.Next(&dbinfo) {
		if dbinfo.Analyzed {
			cleanAnalysis(dbinfo.Name)
			change := mgo.Change{
				Update:    bson.M{"$set": bson.M{"analyzed": false}},
				ReturnNew: true,
			}
			_, err := coll.Find(bson.M{"_id": dbinfo.ID}).Apply(change, &dbinfo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to update metadb: %s\n", err.Error())
			}
			if dbinfo.Analyzed {
				fmt.Fprintf(os.Stderr, "Warning %s may not have updated in meta.\n", dbinfo.Name)
			}
		}
	}
}

// showBeacons shows all beacons for a given database
func showBeacons(dataset string) {

	cols := "score\tsource\tdest\trange\tsize\trange-vals\tfill\tspread\tsum\ttop-interval\t"
	cols += "top-interval-cnt\n"
	tmpl := "{{.Score}}\t{{.Src}}\t{{.Dst}}\t{{.Range}}\t{{.Size}}\t{{.RangeVals}}\t{{.Fill}}"
	tmpl += "\t{{.Spread}}\t{{.Sum}}\t{{.TopInterval}}\t{{.TopIntervalCt}}\n"

	out, err := template.New("tbd").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	conf := config.InitConfig("")
	conf.System.DB = dataset

	coll := conf.Session.DB(dataset).C(conf.System.TBDConfig.TBDTable)
	iter := coll.Find(nil).Iter()

	var res TBD.TBD

	fmt.Printf(cols)
	for iter.Next(&res) {
		err := out.Execute(os.Stdout, res)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}
}

// showScans prints all scans for a given database
func showScans(dataset string) {

	cols := "source\tdest\tport-count\tports\n"
	tmpl := "{{.Src}}\t{{.Dst}}\t{{.PortSetCount}}\t"
	tmpl += "{{range $idx, $port := .PortSet}} {{ $port }} {{end}}\n"
	out, err := template.New("scn").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	conf := config.InitConfig("")
	conf.System.DB = dataset

	var res scanning.Scan

	coll := conf.Session.DB(dataset).C(conf.System.ScanningConfig.ScanTable)
	iter := coll.Find(nil).Iter()

	fmt.Printf(cols)
	for iter.Next(&res) {
		err := out.Execute(os.Stdout, res)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}

}

// showBlacklisted prints all blacklisted for a given database
func showBlacklisted(dataset string) {

	cols := "host\tscore\n"
	tmpl := "{{.Host}}\t{{.Score}}\n"
	out, err := template.New("bl").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	conf := config.InitConfig("")
	conf.System.DB = dataset

	var res scanning.Scan

	coll := conf.Session.DB(dataset).C(conf.System.BlacklistedConfig.BlacklistTable)
	iter := coll.Find(nil).Iter()

	fmt.Printf(cols)
	for iter.Next(&res) {
		err := out.Execute(os.Stdout, res)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR: Template failure: %s\n", err.Error())
		}
	}

}
