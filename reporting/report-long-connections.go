package reporting

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
	"github.com/urfave/cli"
)

func printLongConns(db string, res *resources.Resources) error {
	f, err := os.Create("long-conns.html")
	if err != nil {
		return err
	}
	defer f.Close()
	out, err := template.New("long-conns.html").Parse(templates.LongConnsTempl)
	if err != nil {
		return err
	}

	res.DB.SelectDB(db)

	sortStr := "maxdur"
	sortDirection := -1
	thresh := 60 // 1 minute

	data := getLongConnsResultsView(res, thresh, sortStr, sortDirection, 1000)

	w, err := getLongConnWriter(data)
	if err != nil {
		return err
	}
	if len(w) == 0 {
		return cli.NewExitError("No results were found for " + db, -1)
	}
	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getLongConnWriter(conns []uconn.LongConnAnalysisView) (string, error) {
	tmpl := "<tr><td>{{.Src}}</td><td>{{.Dst}}</td><td>{{.TupleStr}}</td><td>{{.MaxDuration}}</td></tr>\n"
	out, err := template.New("Conn").Parse(tmpl)
	if err != nil {
		return "", err
	}
	w := new(bytes.Buffer)
	for _, conn := range conns {
		conn.TupleStr = strings.Join(conn.Tuples, ",  ")
		err := out.Execute(w, conn)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}

//getLongConnsResultsView gets the long connection results
func getLongConnsResultsView(res *resources.Resources, thresh int, sort string, sortDirection int, limit int) []uconn.LongConnAnalysisView {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var longConnResults []uconn.LongConnAnalysisView

	longConnQuery := []bson.M{
		bson.M{"$match": bson.M{"dat.maxdur": bson.M{"$gt": thresh}}},
		bson.M{"$project": bson.M{"maxdur": "$dat.maxdur", "src": "$src", "dst": "$dst", "tuples": bson.M{"$ifNull": []interface{}{"$dat.tuples", []interface{}{}}}}},
		bson.M{"$unwind": "$maxdur"},
		bson.M{"$unwind": "$tuples"},
		bson.M{"$unwind": "$tuples"}, // not an error, must be done twice
		bson.M{"$group": bson.M{
			"_id":    "$_id",
			"maxdur": bson.M{"$max": "$maxdur"},
			"src":    bson.M{"$first": "$src"},
			"dst":    bson.M{"$first": "$dst"},
			"tuples": bson.M{"$addToSet": "$tuples"},
		}},
		bson.M{"$project": bson.M{
			"maxdur": 1,
			"src":    1,
			"dst":    1,
			"tuples": bson.M{"$slice": []interface{}{"$tuples", 5}},
		}},
		bson.M{"$sort": bson.M{sort: sortDirection}},
		bson.M{"$limit": limit},
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.UniqueConnTable).Pipe(longConnQuery).AllowDiskUse().All(&longConnResults)

	if err != nil {
		//TODO: properly log this error
		fmt.Println(err)
	}

	return longConnResults

}
