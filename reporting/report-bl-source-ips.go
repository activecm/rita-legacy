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

func printBLSourceIPs(db string, showNetNames bool, res *resources.Resources, logsGeneratedAt string) error {
	f, err := os.Create("bl-source-ips.html")
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := blacklist.SrcIPResults(res, "conn_count", 1000, false)
	if err != nil {
		return err
	}

	var blSourceIPTempl string
	if showNetNames {
		blSourceIPTempl = templates.BLSourceIPNetNamesTempl
	} else {
		blSourceIPTempl = templates.BLSourceIPTempl
	}

	out, err := template.New("bl-source-ips.html").Parse(blSourceIPTempl)
	if err != nil {
		return err
	}

	w, err := getBLIPWriter(data, showNetNames)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w), LogsGeneratedAt: logsGeneratedAt})
}

func getBLIPWriter(results []blacklist.IPResult, showNetNames bool) (string, error) {
	var tmpl string
	if showNetNames {
		tmpl = "<tr><td>{{.Host.IP}}</td><td>{{.Host.NetworkName}}</td><td>{{.Connections}}</td><td>{{.UniqueConnections}}</td>" +
			"<td>{{.TotalBytes}}</td>" +
			"<td>{{range $idx, $host := .ConnectedHostStrs}}{{if $idx}}, {{end}}{{ $host }}{{end}}</td>" +
			"</tr>\n"
	} else {
		tmpl = "<tr><td>{{.Host.IP}}</td><td>{{.Connections}}</td><td>{{.UniqueConnections}}</td>" +
			"<td>{{.TotalBytes}}</td>" +
			"<td>{{range $idx, $host := .ConnectedHostStrs}}{{if $idx}}, {{end}}{{ $host }}{{end}}</td>" +
			"</tr>\n"
	}

	out, err := template.New("blip").Parse(tmpl)
	if err != nil {
		return "", err
	}

	w := new(bytes.Buffer)

	for _, result := range results {

		//format UniqueIP destinations
		var connectedHostStrs []string
		for _, connectedUniqIP := range result.Peers {

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
			blacklist.IPResult
			ConnectedHostStrs []string
		}{result, connectedHostStrs}

		err := out.Execute(w, formattedResult)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}
