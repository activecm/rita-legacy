package beacon

import (
	"fmt"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/ocmdev/rita/database"
	datatype_beacon "github.com/ocmdev/rita/datatypes/beacon"
)

func printAnalysis(res *datatype_beacon.BeaconAnalysisOutput) string {
	v := reflect.ValueOf(*res)

	var ret string
	ret += "\n"

	for i := 0; i < v.NumField(); i++ {
		ret += fmt.Sprintf("\t%s:\t%v\n", v.Type().Field(i).Name, v.Field(i).Interface())
	}

	return ret
}

func TestAnalysis(t *testing.T) {
	resources := database.InitMockResources("")
	resources.Log.Level = log.DebugLevel
	resources.System.BeaconConfig.DefaultConnectionThresh = 2

	fail := false
	for i, val := range testDataList {
		beaconing := newBeacon(resources)
		//set first and last connection times
		beaconing.minTime = val.ts[0]
		beaconing.maxTime = val.ts[len(val.ts)-1]
		data := &beaconAnalysisInput{
			src: "0.0.0.0",
			dst: "0.0.0.0",
			ts:  val.ts,
		}

		beaconing.analysisWg.Add(1)
		go beaconing.analyze()
		beaconing.analysisChannel <- data
		close(beaconing.analysisChannel)
		res := <-beaconing.writeChannel
		beaconing.analysisWg.Wait()

		status := "PASS"
		if res.TS_score < val.minScore || res.TS_score > val.maxScore {
			fail = true
			status = "FAIL"
		}

		t.Logf("%d - %s:\n\tExpected Score: %f < x < %f\n\tDescription: %s\n%s\n", i, status, val.minScore, val.maxScore, val.description, printAnalysis(res))
	}

	if fail {
		t.Fail()
	}
}
