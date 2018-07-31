package reporting

import (
	"bytes"
	"html/template"
	"os"
	"sort"

	"github.com/globalsign/mgo/bson"

	"github.com/activecm/rita/analysis/dns"
	"github.com/activecm/rita/datatypes/blacklist"
	"github.com/activecm/rita/datatypes/structure"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
)

func printBLHostnames(db string, res *resources.Resources) error {
	f, err := os.Create("bl-hostnames.html")
	if err != nil {
		return err
	}
	defer f.Close()

	var blHosts []blacklist.BlacklistedHostname
	res.DB.Session.DB(db).
		C(res.Config.T.Blacklisted.HostnamesTable).
		Find(nil).Sort("-conn").All(&blHosts)

	//for each blacklisted host
	for i, host := range blHosts {
		//get the ips associated with the host
		ips := dns.GetIPsFromHost(res, host.Hostname)
		//and loop over the ips
		for _, ip := range ips {
			//then find all of the hosts which talked to the ip
			var connected []structure.UniqueConnection
			res.DB.Session.DB(db).
				C(res.Config.T.Structure.UniqueConnTable).Find(
				bson.M{"dst": ip},
			).All(&connected)
			//and aggregate the source ip addresses
			for _, uconn := range connected {
				blHosts[i].ConnectedHosts = append(blHosts[i].ConnectedHosts, uconn.Src)
			}
		}
	}

	out, err := template.New("bl-hostnames.html").Parse(templates.BLHostnameTempl)
	if err != nil {
		return err
	}

	w, err := getBLHostnameWriter(blHosts)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getBLHostnameWriter(results []blacklist.BlacklistedHostname) (string, error) {
	tmpl := "<tr><td>{{.Hostname}}</td><td>{{.Connections}}</td><td>{{.UniqueConnections}}</td>" +
		"<td>{{.TotalBytes}}</td>" +
		"<td>{{range $idx, $list := .Lists}}{{if $idx}}, {{end}}{{ $list }}{{end}}</td>" +
		"<td>{{range $idx, $host := .ConnectedHosts}}{{if $idx}}, {{end}}{{ $host }}{{end}}</td>" +
		"</tr>\n"

	out, err := template.New("blhostname").Parse(tmpl)
	if err != nil {
		return "", err
	}

	w := new(bytes.Buffer)

	for _, result := range results {
		sort.Strings(result.Lists)
		sort.Strings(result.ConnectedHosts)
		err := out.Execute(w, result)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}
