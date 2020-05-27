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
)

// PrintHTML is the primary html Print function, this command takes in a
// list of databases and the resource object from the main rita program and
// will use HTML templating to write out the results of `rita analyze` into
// a directory named after the selected dataset, or `rita-html-report` if
// mupltiple were selected, within the current working directory,
// mongodb must be running to call this command, will exit on any writing error
func PrintHTML(dbsIn []string, dir string, res *resources.Resources) error {
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
	// Build directory string
	outFolder = append([]byte(dir), outFolder...)
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
			return err
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

func writeDBHomePage(db string) error {
	f, err := os.Create("index.html")
	if err != nil {
		return err
	}
	defer f.Close()

	out, err := template.New("index.html").Parse(htmlTempl.DBhometempl)
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

	err = writeDBHomePage(db)
	if err != nil {
		fmt.Println("[-] Error writing Home page: " + err.Error())
	}

	err = printDNS(db, res)
	if err != nil {
		fmt.Println("[-] Error writing DNS page: " + err.Error())
	}
	err = printBLSourceIPs(db, res)
	if err != nil {
		fmt.Println("[-] Error writing blacklist-source page: " + err.Error())
	}
	err = printBLDestIPs(db, res)
	if err != nil {
		fmt.Println("[-] Error writing blacklist-destination page: " + err.Error())
	}
	err = printBLHostnames(db, res)
	if err != nil {
		fmt.Println("[-] Error writing blacklist-hostnames page: " + err.Error())
	}

	err = printBeacons(db, res)
	if err != nil {
		fmt.Println("[-] Error writing beacons page: " + err.Error())
	}

	err = printStrobes(db, res)
	if err != nil {
		fmt.Println("[-] Error writing strobes page: " + err.Error())
	}

	err = printLongConns(db, res)
	if err != nil {
		fmt.Println("[-] Error writing long connections page: " + err.Error())
	}
	err = printUserAgents(db, res)
	if err != nil {
		fmt.Println("[-] Error writing user agents page: " + err.Error())
	}

	err = os.Chdir("..")
	if err != nil {
		fmt.Println("[-] Error changing to home directory, but if it got here all the pages are probably written: " + err.Error())
	}

	return nil
}
