package printing

import (
	"bytes"
	"html/template"
	"os"

	"github.com/bglebrun/rita/analysis/blacklisted"
	"github.com/bglebrun/rita/database"
	blacklistedData "github.com/bglebrun/rita/datatypes/blacklisted"
	htmlTempl "github.com/bglebrun/rita/printing/templates"
)

func printBlacklisted(db string, res *database.Resources) error {
	res.DB.SelectDB(db)

	var result blacklistedData.Blacklist
	var results []blacklistedData.Blacklist

	coll := res.DB.Session.DB(db).C(res.System.BlacklistedConfig.BlacklistTable)
	iter := coll.Find(nil).Sort("-count").Iter()

	for iter.Next(&result) {
		blacklisted.SetBlacklistSources(res, &result)
		results = append(results, result)
	}

	return printBlacklistedHTML(results, db)
}

// printBlacklistedHTML prints all blacklisted for a given database
func printBlacklistedHTML(results []blacklistedData.Blacklist, db string) error {

	f, err := os.Create("blacklisted.html")
	if err != nil {
		return err
	}
	defer f.Close()

	out, err := template.New("blacklisted.html").Parse(htmlTempl.BlacklistedTempl)
	if err != nil {
		return err
	}
	w, err := getBlacklistWriter(results)
	if err != nil {
		return err
	}
	return out.Execute(f, &scan{Dbs: db, Writer: template.HTML(w)})
}

func getBlacklistWriter(results []blacklistedData.Blacklist) (string, error) {
	tmpl := "<tr><td>{{.Host}}</td><td>{{.Score}}</td><td>{{range $idx, $src := .Sources}}{{if $idx}}{{end}}{{$src}}{{end}}</td></tr>\n"
	w := new(bytes.Buffer)
	out, err := template.New("blacklist").Parse(tmpl)
	if err != nil {
		return "", err
	}

	for _, result := range results {
		err := out.Execute(w, result)
		if err != nil {
			return "", err
		}
	}

	return w.String(), nil
}
