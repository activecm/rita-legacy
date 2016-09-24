package server

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/mgo.v2"
)

type (
	// RitaServer provides access to server components and is inteded to
	// reduce the complexity of dealing with the backend by allowing a
	// pluggable front end.
	RitaServer struct {
		Address string         // Default 172.0.0.1:8080
		tTable  *templateTable // template support
		log     *log.Logger    // logging
		ssn     *mgo.Session   // address of the database
	}

	// templateTable is meant to reduce complexity of templating operations
	templateTable struct {
		t map[string]*template.Template // template mapping
	}
)

// New creates a new server object at address
func New(address, templates, dbms string) *RitaServer {

	lg := log.New(os.Stderr, "RITA-FE: ", log.Ldate|log.Ltime|log.Lshortfile)
	t := newTemplateTable(templates+"/templates", lg)

	lg.Println("Connecting to database: ", dbms)
	ssn, err := mgo.Dial(dbms)
	if err != nil {
		panic(err)
	}

	lg.Println("Connection success, proceeding")
	return &RitaServer{
		Address: address,
		tTable:  t,
		ssn:     ssn,
		log:     lg,
	}
}

// newTemplateTable initializes our template table
func newTemplateTable(templateDir string, lg *log.Logger) *templateTable {

	files, err := ioutil.ReadDir(templateDir)
	lg.Println("Building template table")
	if err != nil {
		panic(err)
	}

	templs := make(map[string]*template.Template)

	basepath := templateDir + "/" + "base.html"

	for _, finfo := range files {

		// dont build a template of the base template agianst itself
		if finfo.Name() == "base.html" {
			continue
		}

		// skip anything that's not labeled html in the templates dir
		if !strings.HasSuffix(finfo.Name(), "html") {
			continue
		}

		lg.Println("Adding entry for", finfo.Name())
		path := templateDir + "/" + finfo.Name()

		templs[finfo.Name()] = template.Must(template.ParseFiles(path, basepath))
	}

	return &templateTable{t: templs}
}

// Start launches the server
func (rs *RitaServer) Start() {
	rs.log.Println("Constructing handlers")
	http.HandleFunc("/", rs.indexRoute)

	rs.log.Println("Starting listener on: ", rs.Address)
	rs.log.Fatal(http.ListenAndServe(rs.Address, nil))
}

// Hanlde conenctions to the index page, should display a page offering up the
// option to either start a new test, or select a previously built test from the
// database and work with that.
func (rs *RitaServer) indexRoute(w http.ResponseWriter, r *http.Request) {

	var page struct {
		Title  string
		Tests  []string
		HasDbs bool
	}
	page.Title = "Welcome to RITA"
	page.Tests = rs.getAllMetaDB()

	if len(page.Tests) == 0 {
		page.HasDbs = false
	} else {
		page.HasDbs = true
	}

	rs.tTable.t["index.html"].ExecuteTemplate(w, "base", page)
}

// getAllMetaDB essentially returns the databases that look like metadbs
func (rs *RitaServer) getAllMetaDB() []string {
	s := rs.ssn.Copy()
	defer s.Close()

	var result []string

	alldb, err := s.DatabaseNames()
	if err != nil {
		panic(err)
	}

	for _, db := range alldb {
		var files bool
		var databases bool
		colns, err := s.DB(db).CollectionNames()
		if err != nil {
			continue
		}
		for _, name := range colns {
			if name == "files" {
				files = true
			}
			if name == "databases" {
				databases = true
			}
		}

		if files && databases {
			result = append(result, db)
		}
	}

	return result
}
