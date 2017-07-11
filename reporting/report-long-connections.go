package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/data"
	"github.com/ocmdev/rita/reporting/templates"
)

func printLongConns(db string, res *database.Resources) error {
	f, err := os.Create("long-conns.html")
	if err != nil {
		return err
	}
	defer f.Close()
	out, err := template.New("long-conns.html").Parse(templates.LongConnsTempl)
	if err != nil {
		return err
	}

	var conns []data.Conn
	coll := res.DB.Session.DB(db).C(res.Config.T.Structure.ConnTable)
	coll.Find(nil).Sort("-duration").Limit(1000).All(&conns)

	w, err := getLongConnWriter(conns)
	if err != nil {
		return err
	}
	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getLongConnWriter(conns []data.Conn) (string, error) {
	tmpl := "<tr><td>{{.Src}}</td><td>{{.Spt}}</td><td>{{.Dst}}</td><td>{{.Dpt}}</td><td>{{.Dur}}</td><td>{{.Proto}}</td></tr>\n"
	out, err := template.New("Conn").Parse(tmpl)
	if err != nil {
		return "", err
	}
	w := new(bytes.Buffer)
	for _, conn := range conns {
		err := out.Execute(w, conn)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}
