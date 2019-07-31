package reporting

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"strconv"

	htmlTempl "github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
	"github.com/activecm/rita/util"
	"github.com/skratchdot/open-golang/open"
	"github.com/urfave/cli"
)

// PrintHTML is the primary html Print function, this command takes in a
// list of databases and the resource object from the main rita program and
// will use HTML templating to write out the results of `rita analyze` into
// a directory named after the selected dataset, or `rita-html-report` if
// mupltiple were selected, within the current working directory,
// mongodb must be running to call this command, will exit on any writing error
func PrintHTML(dbsIn []string, res *resources.Resources) error {
	if len(dbsIn) == 0 {
		return errors.New("no analyzed databases to report on")
	}

	var dbs []string
	for _, db := range dbsIn {
		dbs = append(dbs, db)
	}
	if len(dbs) == 0 {
		return errors.New("none of the selected databases have been analyzed")
	}

	//create outFolder as our string builder
	var outFolder []byte
	if len(dbs) == 1 {
		outFolder = []byte(dbs[0])
	} else {
		outFolder = []byte("rita-html-report")
	}
	outFolderBaseLen := len(outFolder)
	counter := 1

	//while the file exists, append the next counter
	for _, err := os.Stat(string(outFolder)); err == nil; _, err = os.Stat(string(outFolder)) {
		outFolder = outFolder[:outFolderBaseLen]
		outFolder = append(outFolder, []byte(strconv.Itoa(counter))...)
		counter++
	}
	outFolderString := string(outFolder)

	err := os.Mkdir(outFolderString, 0755)

	if err != nil {
		return (err)
	}

	os.Chdir(outFolderString)

	// First, print our home page with our databases, pointing to each db
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Write the homepage
	err = writeHomePage(dbs)
	if err != nil {
		return err
	}

	// Start db iteration
	for k := range dbs {
		err = writeDB(dbs[k], wd, res)
		if err != nil {
			if err.Error() == ("No results were found for " + dbs[k]) {
				fmt.Println("[!] " + err.Error())
				fmt.Println("    This might indicate that this database has been dropped, but the metadatabase hasn't been updated.")
				fmt.Println("    To update the metadatabase and fix this problem, run:")
				fmt.Println("    rita delete-database " + dbs[k])
			} else {
				return err
			}
		}
	}

	fmt.Println("[-] Wrote outputs, check " + wd + " for files")
	os.Chdir("..")
	open.Run("./" + outFolderString + "/index.html")
	// End db iteration
	return nil
}

func writeHomePage(Dbs []string) error {
	f, err := os.Create("index.html")
	if err != nil {
		return err
	}
	defer f.Close()

	err = ioutil.WriteFile("github.svg", htmlTempl.GithubSVG, 0644)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("style.css", htmlTempl.CSStempl, 0644)
	if err != nil {
		return err
	}

	out, err := template.New("home.html").Parse(htmlTempl.Hometempl)
	if err != nil {
		return err
	}
	return out.Execute(f, Dbs)
}

func writeDBHomePage(db string, empty bool) error {
	f, err := os.Create("index.html")
	if err != nil {
		return err
	}
	defer f.Close()

	templ := htmlTempl.DBhometempl
	if empty {
		templ = htmlTempl.DBemptyhometempl
	}

	out, err := template.New("index.html").Parse(templ)
	if err != nil {
		return err
	}

	return out.Execute(f, htmlTempl.ReportingInfo{DB: db})
}

func writeDB(db string, wd string, res *resources.Resources) error {
	writeDir := wd + "/" + db
	var err error

	fmt.Print("[-] Writing: " + writeDir + "\n")
	if !util.Exists(writeDir) {
		err = os.Mkdir(db, 0755)
		if err != nil {
			return err
		}
		err = os.Chdir(db)
		if err != nil {
			return err
		}
	}
	res.DB.SelectDB(db)

	hasResults := false

	err = writeDBHomePage(db, false)
	if err != nil {
		return err
	}

	err = printDNS(db, res)
	if err == nil {
		hasResults = true
	} else if err.Error() != ("No results were found for " + db) {
		// if we have an error other than no results return it
		return err
	}

	err = printBLSourceIPs(db, res)
	if err == nil {
		hasResults = true
	} else if err.Error() != ("No results were found for " + db) {
		// if we have an error other than no results return it
		return err
	}

	err = printBLDestIPs(db, res)
	if err == nil {
		hasResults = true
	} else if err.Error() != ("No results were found for " + db) {
		// if we have an error other than no results return it
		return err
	}

	err = printBLHostnames(db, res)
	if err == nil {
		hasResults = true
	} else if err.Error() != ("No results were found for " + db) {
		// if we have an error other than no results return it
		return err
	}

	err = printBeacons(db, res)
	if err == nil {
		hasResults = true
	} else if err.Error() != ("No results were found for " + db) {
		// if we have an error other than no results return it
		return err
	}

	err = printStrobes(db, res)
	if err == nil {
		hasResults = true
	} else if err.Error() != ("No results were found for " + db) {
		// if we have an error other than no results return it
		return err
	}

	err = printLongConns(db, res)
	if err == nil {
		hasResults = true
	} else if err.Error() != ("No results were found for " + db) {
		// if we have an error other than no results return it
		return err
	}

	err = printUserAgents(db, res)
	if err == nil {
		hasResults = true
	} else if err.Error() != ("No results were found for " + db) {
		// if we have an error other than no results return it
		return err
	}


	// only return "no results" error if none of the modules had results
	var resultsErr error
	if !hasResults {
		os.Remove("index.html")

		err = writeDBHomePage(db, true)
		if err != nil {
			return err
		}

		resultsErr = cli.NewExitError("No results were found for " + db, -1)
	}

	err = os.Chdir("..")
	if err != nil {
		return err
	}

	return resultsErr
}
