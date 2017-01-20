package TBD

import (
	"fmt"
	"reflect"
	"testing"

	datatype_TBD "github.com/ocmdev/rita/datatypes/TBD"
)

func printAnalysis(res datatype_TBD.TBDAnalysisOutput) string {
	v := reflect.ValueOf(res)

	var ret string
	ret += "\n"

	for i := 0; i < v.NumField(); i++ {
		ret += fmt.Sprintf("\t%s:\t%v\n", v.Type().Field(i).Name, v.Field(i).Interface())
	}

	return ret
}

func TestAnalysis(t *testing.T) {
	fail := false
	for i, val := range testDataList {
		var data tbdAnalysisInput
		data.src = "0.0.0.0"
		data.dst = "0.0.0.0"
		data.ts = val.ts
		res, _ := (analysis(&data, 2, val.maxTime, val.minTime))

		status := "PASS"
		if res.TS_score < val.scoreThresh {
			fail = true
			status = "FAIL"
		}

		t.Logf("%d - %s:\n\tExpected Score: >%f\n\tDescription: %s\n%s\n", i, status, val.scoreThresh, val.description, printAnalysis(res))

	}

	if fail {
		t.Fail()
	}
}
