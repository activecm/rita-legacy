package printing

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"

	"github.com/bglebrun/rita/config"
	"github.com/bglebrun/rita/database"
	htmlTempl "github.com/bglebrun/rita/printing/templates"
	mgo "gopkg.in/mgo.v2"
)

// Printing is our main printing function
func Printing(dbs []string, res *database.Resources) error {
	con, ok := config.GetConfig("")
	if !ok {
		return errors.New("unable to get config")
	}

	host := con.DatabaseHost

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

	fmt.Println("[-] Wrote outputs, check " + wd + " for files")
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

	err = ioutil.WriteFile("style.css", htmlTempl.CSStempl, 0777)
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
	fExists, err := exists(writeDir)
	if err != nil {
		return err
	}
	if !fExists {
		err = os.Mkdir(db, 0777)
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
	err = printDNSHtml(db, res)
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
