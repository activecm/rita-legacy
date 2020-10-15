package reporting

import (
	"bytes"
	"html/template"
	"os"
	//TODO[AGENT]: Sort UniqIPs
	//"sort"

	"github.com/activecm/rita/pkg/blacklist"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
)

func printBLHostnames(db string, res *resources.Resources) error {
	f, err := os.Create("bl-hostnames.html")
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := blacklist.HostnameResults(res, "conn_count", 1000, false)
	if err != nil {
		return err
	}

	out, err := template.New("bl-hostnames.html").Parse(templates.BLHostnameTempl)
	if err != nil {
		return err
	}

	w, err := getBLHostnameWriter(data)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getBLHostnameWriter(results []blacklist.HostnameResult) (string, error) {
	tmpl := "<tr><td>{{.Host}}</td><td>{{.Connections}}</td><td>{{.UniqueConnections}}</td>" +
		"<td>{{.TotalBytes}}</td>" +
		"<td>{{range $idx, $host := .ConnectedHosts}}{{if $idx}}, {{end}}{{ $host }}{{end}}</td>" +
		"</tr>\n"

	out, err := template.New("blhostname").Parse(tmpl)
	if err != nil {
		return "", err
	}

	w := new(bytes.Buffer)

	for _, result := range results {
		//TODO[AGENT]: Sort UniqIPs
		//sort.Strings(result.ConnectedHosts)
		err := out.Execute(w, result)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}
