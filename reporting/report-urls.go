package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/urls"
	"github.com/ocmdev/rita/reporting/templates"
)

func printLongURLs(db string, res *database.Resources) error {
	f, err := os.Create("long-urls.html")
	if err != nil {
		return err
	}
	defer f.Close()
	out, err := template.New("long-urls.html").Parse(templates.LongURLsTempl)
	if err != nil {
		return err
	}

	var urls []urls.URL
	coll := res.DB.Session.DB(db).C(res.Config.T.Urls.UrlsTable)
	coll.Find(nil).Sort("-length").Limit(1000).All(&urls)

	w, err := getLongURLWriter(urls)
	if err != nil {
		return err
	}
	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getLongURLWriter(urls []urls.URL) (string, error) {
	tmpl := "<tr><td>{{.URL}}</td><td>{{.URI}}</td><td>{{.Length}}</td><td>{{.Count}}</td></tr>\n"
	out, err := template.New("Urls").Parse(tmpl)
	if err != nil {
		return "", err
	}
	w := new(bytes.Buffer)
	for _, url := range urls {
		if len(url.URL) > 50 {
			url.URL = url.URL[0:47] + "..."
		}
		if len(url.URI) > 50 {
			url.URI = url.URI[0:47] + "..."
		}
		err := out.Execute(w, url)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}
