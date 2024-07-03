package reporting

import (
	"bytes"
	"html/template"
	"os"

	"github.com/activecm/rita-legacy/pkg/beaconproxy"
	"github.com/activecm/rita-legacy/reporting/templates"
	"github.com/activecm/rita-legacy/resources"
)

func printBeaconsProxy(db string, showNetNames bool, res *resources.Resources, logsGeneratedAt string) error {
	var w string
	f, err := os.Create("beaconsproxy.html")
	if err != nil {
		return err
	}
	defer f.Close()

	var beaconsProxyTempl string
	if showNetNames {
		beaconsProxyTempl = templates.BeaconsProxyNetNamesTempl
	} else {
		beaconsProxyTempl = templates.BeaconsProxyTempl
	}

	out, err := template.New("beaconproxy.html").Parse(beaconsProxyTempl)
	if err != nil {
		return err
	}

	data, err := beaconproxy.Results(res, 0)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		w = ""
	} else {
		w, err = getBeaconProxyWriter(data, showNetNames)
		if err != nil {
			return err
		}
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w), LogsGeneratedAt: logsGeneratedAt})
}

func getBeaconProxyWriter(beaconsProxy []beaconproxy.Result, showNetNames bool) (string, error) {
	tmpl := "<tr>"

	tmpl += "<td>{{printf \"%.3f\" .Score}}</td>"

	if showNetNames {
		tmpl += "<td>{{.SrcNetworkName}}</td>"
	}

	tmpl += "<td>{{.SrcIP}}</td><td>{{.FQDN}}</td>"

	if showNetNames {
		tmpl += "<td>{{.Proxy.NetworkName}}</td>"
	}

	tmpl += "<td>{{.Proxy.IP}}</td>"

	tmpl += "<td>{{.Connections}}</td><td>{{printf \"%.3f\" .Ts.Score}}</td>"
	tmpl += "<td>{{printf \"%.3f\" .DurScore}}</td><td>{{printf \"%.3f\" .HistScore}}</td><td>{{.Ts.Mode}}</td>"
	tmpl += "</tr>\n"

	out, err := template.New("beaconproxy").Parse(tmpl)
	if err != nil {
		return "", err
	}

	w := new(bytes.Buffer)

	for _, result := range beaconsProxy {
		err = out.Execute(w, result)
		if err != nil {
			return "", err
		}
	}

	return w.String(), nil
}
