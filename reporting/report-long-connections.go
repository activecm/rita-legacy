package reporting

import (
	"bytes"
	"html/template"
	"os"
	"strings"

	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
)

func printLongConns(db string, showNetNames bool, res *resources.Resources, logsGeneratedAt string) error {
	f, err := os.Create("long-conns.html")
	if err != nil {
		return err
	}
	defer f.Close()

	var longConnsTempl string
	if showNetNames {
		longConnsTempl = templates.LongConnsNetNamesTempl
	} else {
		longConnsTempl = templates.LongConnsTempl
	}

	out, err := template.New("long-conns.html").Parse(longConnsTempl)
	if err != nil {
		return err
	}

	res.DB.SelectDB(db)

	thresh := 60 // 1 minute
	data, err := uconn.LongConnResults(res, thresh, 1000, false)
	if err != nil {
		return err
	}

	w, err := getLongConnWriter(data, showNetNames)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w), LogsGeneratedAt: logsGeneratedAt})
}

func getLongConnWriter(conns []uconn.LongConnResult, showNetNames bool) (string, error) {
	var tmpl string
	if showNetNames {
		tmpl = "<tr><td>{{.SrcNetworkName}}</td><td>{{.DstNetworkName}}</td><td>{{.SrcIP}}</td><td>{{.DstIP}}</td><td>{{.TupleStr}}</td><td>{{.MaxDuration}}</td></tr>\n"
	} else {
		tmpl = "<tr><td>{{.SrcIP}}</td><td>{{.DstIP}}</td><td>{{.TupleStr}}</td><td>{{.MaxDuration}}</td></tr>\n"
	}

	out, err := template.New("Conn").Parse(tmpl)
	if err != nil {
		return "", err
	}
	w := new(bytes.Buffer)
	for _, conn := range conns {
		connTmplData := struct {
			uconn.LongConnResult
			TupleStr string
		}{conn, strings.Join(conn.Tuples, ",  ")}

		err := out.Execute(w, connTmplData)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}
