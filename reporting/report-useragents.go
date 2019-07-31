package reporting

import (
	"bytes"
	"fmt"
	"html/template"
	"os"

	"github.com/activecm/rita/pkg/useragent"
	"github.com/activecm/rita/reporting/templates"
	"github.com/activecm/rita/resources"
	"github.com/globalsign/mgo/bson"
	"github.com/urfave/cli"
)

func printUserAgents(db string, res *resources.Resources) error {
	f, err := os.Create("useragents.html")
	if err != nil {
		return err
	}
	defer f.Close()
	out, err := template.New("useragents.html").Parse(templates.UserAgentsTempl)
	if err != nil {
		return err
	}

	data := getUseragentResultsView(res, "seen", 1, 1000)

	w, err := getUserAgentsWriter(data)
	if err != nil {
		return err
	}
	if len(w) == 0 {
		return cli.NewExitError("No results were found for " + db, -1)
	}
	return out.Execute(f, &templates.ReportingInfo{DB: db, Writer: template.HTML(w)})
}

func getUserAgentsWriter(agents []useragent.AnalysisView) (string, error) {
	tmpl := "<tr><td>{{.UserAgent}}</td><td>{{.TimesUsed}}</td></tr>\n"
	out, err := template.New("Agents").Parse(tmpl)
	if err != nil {
		return "", err
	}
	w := new(bytes.Buffer)
	for _, agent := range agents {
		err := out.Execute(w, agent)
		if err != nil {
			return "", err
		}
	}
	return w.String(), nil
}

//getUseragentResultsView gets the useragent results
func getUseragentResultsView(res *resources.Resources, sort string, sortDirection int, limit int) []useragent.AnalysisView {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var useragentResults []useragent.AnalysisView

	useragentQuery := []bson.M{
		bson.M{"$project": bson.M{"user_agent": 1, "seen": "$dat.seen"}},
		bson.M{"$unwind": "$seen"},
		bson.M{"$group": bson.M{
			"_id":  "$user_agent",
			"seen": bson.M{"$sum": "$seen"},
		}},
		bson.M{"$project": bson.M{
			"_id":        0,
			"user_agent": "$_id",
			"seen":       1,
		}},
		bson.M{"$sort": bson.M{sort: sortDirection}},
		bson.M{"$limit": limit},
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.UserAgent.UserAgentTable).Pipe(useragentQuery).AllowDiskUse().All(&useragentResults)

	if err != nil {
		//TODO: properly log this error
		fmt.Println(err)
	}

	return useragentResults

}
