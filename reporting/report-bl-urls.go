package reporting

import (
	"bytes"
	"html/template"
	"os"
	"sort"

	"gopkg.in/mgo.v2/bson"

	"github.com/ocmdev/rita/database"
	"github.com/ocmdev/rita/datatypes/blacklist"
	"github.com/ocmdev/rita/datatypes/structure"
	"github.com/ocmdev/rita/datatypes/urls"
	"github.com/ocmdev/rita/reporting/templates"
)

func printBLURLs(db string, res *database.Resources) error {
	f, err := os.Create("bl-urls.html")
	if err != nil {
		return err
	}
	defer f.Close()

	var blURLs []blacklist.BlacklistedURL
	res.DB.Session.DB(db).
		C(res.Config.T.Blacklisted.UrlsTable).
		Find(nil).Sort("-conn").All(&blURLs)

	//for each blacklisted url
	for i, blURL := range blURLs {
		//get the ips associated with the url
		var urlEntry urls.URL
		res.DB.Session.DB(db).C(res.Config.T.Urls.UrlsTable).
			Find(bson.M{"url": blURL.Host, "uri": blURL.Resource}).One(&urlEntry)
		ips := urlEntry.IPs
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
				blURLs[i].ConnectedHosts = append(blURLs[i].ConnectedHosts, uconn.Src)
			}
		}
	}

	out, err := template.New("bl-url.html").Parse(templates.BLURLTempl)
	if err != nil {
		return err
	}

	w, err := getBLURLWriter(blURLs)
	if err != nil {
		return err
	}

	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getBLURLWriter(results []blacklist.BlacklistedURL) (string, error) {
	tmpl := "<tr><td>{{.Host}}</td><td>{{.Resource}}</td><td>{{.Connections}}</td>" +
		"<td>{{.UniqueConnections}}</td><td>{{.TotalBytes}}</td>" +
		"<td>{{range $idx, $list := .Lists}}{{if $idx}}, {{end}}{{ $list }}{{end}}</td>" +
		"<td>{{range $idx, $host := .ConnectedHosts}}{{if $idx}}, {{end}}{{ $host }}{{end}}</td>" +
		"</tr>\n"

	out, err := template.New("blurl").Parse(tmpl)
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
