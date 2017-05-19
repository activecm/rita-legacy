package printing

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"

	"github.com/bglebrun/rita/config"
	"github.com/bglebrun/rita/database"
	"github.com/bglebrun/ritahtmltest/pageTemplates"
	mgo "gopkg.in/mgo.v2"
)

// Printing is our main printing function
func Printing(res *database.Resources) error {
	con, ok := config.GetConfig("")
	if !ok {
		return errors.New("unable to get config")
	}

	dbs := res.MetaDB.GetDatabases()

	host := con.DatabaseHost
	if len(os.Args) > 1 {
		host = os.Args[2]
	}

	session, err := mgo.Dial(host)
	if err != nil {
		return err
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)

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

	fmt.Println("Wrote outputs, check ~/.rita/html for files")
	// End db iteration
	return nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func writeHomePage(Dbs []string) error {
	f, err := os.Create("index.html")
	if err != nil {
		return err
	}
	defer f.Close()

	err = ioutil.WriteFile("style.css", pageTemplates.CSStempl, 0644)
	if err != nil {
		return err
	}

	out, err := template.New("home.html").Parse(pageTemplates.Hometempl)
	if err != nil {
		return err
	}
	return out.Execute(f, Dbs)
}

func writeDB(db string, wd string, res *database.Resources) error {
	writeDir := wd + "/" + db

	fExists, err := exists(writeDir)
	if err != nil {
		return err
	}
	if !fExists {
		os.Mkdir(writeDir, 0644)
	}

	err = printScans(db, writeDir, res)
	if err != nil {
		return err
	}
	err = printBlacklisted(db, writeDir, res)
	if err != nil {
		return err
	}
	err = printDNSHtml(db, writeDir, res)
	if err != nil {
		return err
	}
	err = printBeacons(db, writeDir, res)
	if err != nil {
		return err
	}

	return nil
}
