package TBD

import (
	"fmt"
	"testing"

	datatype_TBD "github.com/ocmdev/rita/datatypes/TBD"
)

func generateBeacon() tbdAnalysisInput {

}

func printAnalysis(res *datatype_TBD.TBDAnalysisOutput) string {
	return fmt.Sprintf("%+v\n", res)
}

func TestAnalysis(t *testing.T) {
	var data tbdAnalysisInput
	data.src = "1.1.1.1"
	data.dst = "2.2.2.2"
	data.ts = []int64{0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60}
	res, _ := (analysis(&data, 2, 60, 0))
	t.Log(printAnalysis(&res))
}

func BenchmarkAnalysis(b *testing.B) {
}
