package reporting

import (
	"bytes"
	"html/template"
	"os"
	"sort"
	"strings"

	"github.com/activecm/rita-legacy/pkg/blacklist"
	"github.com/activecm/rita-legacy/reporting/templates"
	"github.com/activecm/rita-legacy/resources"
)

func printBLHostnames(db string, showNetNames bool, res *resources.Resources, logsGeneratedAt string) error {
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

	w, err := getBLHostnameWriter(data, showNetNames)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w), LogsGeneratedAt: logsGeneratedAt})
}

func getBLHostnameWriter(results []blacklist.HostnameResult, showNetNames bool) (string, error) {
	tmpl := "<tr><td>{{.Host}}</td><td>{{.Connections}}</td><td>{{.UniqueConnections}}</td>" +
		"<td>{{.TotalBytes}}</td>" +
		"<td>{{range $idx, $host := .ConnectedHostStrs}}{{if $idx}}, {{end}}{{ $host }}{{end}}</td>" +
		"</tr>\n"

	out, err := template.New("blhostname").Parse(tmpl)
	if err != nil {
		return "", err
	}

	w := new(bytes.Buffer)

	for _, result := range results {

		//format UniqueIP destinations
		var connectedHostStrs []string
		for _, connectedUniqIP := range result.ConnectedHosts {

			var connectedIPStr string
			if showNetNames {
				escapedNetName := strings.ReplaceAll(connectedUniqIP.NetworkName, " ", "_")
				escapedNetName = strings.ReplaceAll(escapedNetName, ":", "_")
				connectedIPStr = escapedNetName + ":" + connectedUniqIP.IP
			} else {
				connectedIPStr = connectedUniqIP.IP
			}

			connectedHostStrs = append(connectedHostStrs, connectedIPStr)
		}
		sort.Strings(connectedHostStrs)

		formattedResult := struct {
			blacklist.HostnameResult
			ConnectedHostStrs []string
		}{result, connectedHostStrs}

		err := out.Execute(w, formattedResult)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}
