package beacon

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/activecm/rita/database"
	datatype_beacon "github.com/activecm/rita/datatypes/beacon"
	log "github.com/sirupsen/logrus"
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
	//There are mock resources to set our database equal to, so we don't have errors
	//TODO: update the mock.go file under the database to perfectly mock the MongoDB
	res := database.InitMockResources("../../etc/rita.yaml")
	res.Log.Level = log.DebugLevel
	res.Config.S.Beacon.DefaultConnectionThresh = 2

	//Assume that we have succeeded until we have proof that we haven't. . .
	//  If you think about it, that's a good way to live life too. . .
	fail := false
	//Now we want to iterate through all test cases in beacon_test_data.go
	for i, val := range testDataList {
		beaconing := newBeacon(res)
		//set first and last connection times
		beaconing.minTime = val.ts[0]
		beaconing.maxTime = val.ts[len(val.ts)-1]
		//Now fill in the data that we will need to analyze traffic
		data := &beaconAnalysisInput{
			src:           "0.0.0.0",
			dst:           "0.0.0.0",
			ts:            val.ts, //these are the timestamps
			orig_ip_bytes: val.ds, //these are the data sizes
		}

		//set the wait time for the beaconing analysis
		beaconing.analysisWg.Add(1)
		//Open a channel to the analyze function
		go beaconing.analyze()
		//Feed the data into our new channel
		beaconing.analysisChannel <- data
		//Clonse this input channel
		close(beaconing.analysisChannel)
		//Now set our result to the output to our writeChannel
		res := <-beaconing.writeChannel
		//now we wait for the time we specified earlier
		beaconing.analysisWg.Wait()

		//Now we check if we are inside the acceptable score range
		status := "PASS"
		if res.Score < val.minScore || res.Score > val.maxScore {
			fail = true
			status = "FAIL"
		}

		//Print Results
		t.Logf("%d - %s:\n\tExpected Score: %f < x < %f\n\tDescription: %s\n%s\n", i, status, val.minScore, val.maxScore, val.description, printAnalysis(res))
	}

	//Log the test results
	if fail {
		t.Fail()
	}
}
