package reporting

import (
	"bytes"
	"fmt"
	"html/template"
	"os"

	"github.com/activecm/rita/pkg/beacon"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
	"github.com/urfave/cli"
)

func printStrobes(db string, res *resources.Resources) error {
	f, err := os.Create("strobes.html")
	if err != nil {
		return err
	}
	defer f.Close()
	out, err := template.New("strobes.html").Parse(templates.StrobesTempl)
	if err != nil {
		return err
	}

	data := getStrobeResultsView(res, "conn_count", 1000)

	w, err := getStrobesWriter(data)
	if err != nil {
		return err
	}
	if len(w) == 0 {
		return cli.NewExitError("No results were found for " + db, -1)
	}
	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getStrobesWriter(strobes []beacon.StrobeAnalysisView) (string, error) {
	tmpl := "<tr><td>{{.Src}}</td><td>{{.Dst}}</td><td>{{.ConnectionCount}}</td></tr>\n"
	out, err := template.New("Strobes").Parse(tmpl)
	if err != nil {
		return "", err
	}
	w := new(bytes.Buffer)
	for _, strobe := range strobes {
		err := out.Execute(w, strobe)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}

//getStrobeResultsView ...
func getStrobeResultsView(res *resources.Resources, sort string, limit int) []beacon.StrobeAnalysisView {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var strobes []beacon.StrobeAnalysisView

	strobeQuery := []bson.M{
		bson.M{"$match": bson.M{"strobe": true}},
		bson.M{"$unwind": "$dat"},
		bson.M{"$project": bson.M{"src": 1, "dst": 1, "conns": "$dat.count"}},
		bson.M{"$group": bson.M{
			"_id":        "$_id",
			"src":        bson.M{"$first": "$src"},
			"dst":        bson.M{"$first": "$dst"},
			"conn_count": bson.M{"$sum": "$conns"},
		}},
		bson.M{"$sort": bson.M{sort: -1}},
		bson.M{"$limit": limit},
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.UniqueConnTable).Pipe(strobeQuery).AllowDiskUse().All(&strobes)

	if err != nil {
		//TODO: properly log this error
		fmt.Println(err)
	}

	return strobes
}
