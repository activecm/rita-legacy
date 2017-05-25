package reporting

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"

	"github.com/ocmdev/rita/database"
	htmlTempl "github.com/ocmdev/rita/reporting/templates"
	"github.com/ocmdev/rita/util"
	"github.com/skratchdot/open-golang/open"
)

// PrintHTML is the primary html Print function, this command takes in a
// list of databases and the resource object from the main rita program and
// will use HTML templating to write out the results of `rita analyze` into
// a directory `rita-html-report` within the current working directory,
// mongodb must be running to call this command, will exit on any writing error
func PrintHTML(dbs []string, res *database.Resources) error {
	err := os.Mkdir("rita-html-report", 0755)

	if err != nil {
		fmt.Println(err)
		return (err)
	}

	os.Chdir("rita-html-report")

	session := res.DB.Session.Copy()
	defer session.Close()

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
	open.Run("./rita-html-report/index.html")
	// End db iteration
	return nil
}

func writeHomePage(Dbs []string) error {
	f, err := os.Create("index.html")
	if err != nil {
		return err
	}
	defer f.Close()

	err = ioutil.WriteFile("style.css", htmlTempl.CSStempl, 0755)
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

	return out.Execute(f, db)
}

func writeDB(db string, wd string, res *database.Resources) error {
	writeDir := wd + "/" + db

	fmt.Print("[-] Writing: " + writeDir + "\n")
	fExists, err := util.Exists(writeDir)
	if err != nil {
		return err
	}
	if !fExists {
		err = os.Mkdir(db, 0755)
		if err != nil {
			return err
		}
		err = os.Chdir(db)
		if err != nil {
			return err
		}
	}

	err = writeDBHomePage(db)
	if err != nil {
		fmt.Print(err)
		return err
	}
	err = printScans(db, res)
	if err != nil {
		fmt.Print(err)
		return err
	}
	err = printBlacklisted(db, res)
	if err != nil {
		fmt.Print(err)
		return err
	}
	err = printDNS(db, res)
	if err != nil {
		fmt.Print(err)
		return err
	}
	err = printBeacons(db, res)
	if err != nil {
		fmt.Print(err)
		return err
	}

	err = os.Chdir("..")
	if err != nil {
		fmt.Print(err)
		return err
	}

	return nil
}
