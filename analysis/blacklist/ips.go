package blacklist

import (
	"unsafe"

	"github.com/ocmdev/rita-blacklist2/list"
	"gopkg.in/mgo.v2/bson"

	bl "github.com/ocmdev/rita-blacklist2"
	mgo "gopkg.in/mgo.v2"
)

type ipAggregateResult struct {
	IP string `bson:"ip"`
}

func getUniqueIPFromUconnPipeline(field string) []bson.D {
	return []bson.D{
		{
			{"$project", bson.D{
				{"ip", "$" + field},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", bson.D{
					{"ip", "$ip"},
				}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"ip", "$_id.ip"},
			}},
		},
	}
}

func buildBlacklistedSourceIPs(sourceIPs *mgo.Iter, ssnToCopy *mgo.Session,
	blHandle *bl.Blacklist, destCollection string, bufferSize int) {
	//create session to write to
	ssn := ssnToCopy.Copy()
	defer ssn.Close()

	//create type for communicating rita-bl results
	resultsChannel := make(resultsChan)

	go checkRitaBlacklistIPs(sourceIPs, blHandle, bufferSize, resultsChannel)

	for results := range resultsChannel {
		for ipAddr, individualResults := range results {
			if len(individualResults) > 0 {
				_ = ipAddr
				//store the results
			}
		}
	}
}

func buildBlacklistedDestIPs(destIPs *mgo.Iter, ssnToCopy *mgo.Session,
	blHandle *bl.Blacklist, destCollection string, bufferSize int) {
	//create session to write to
	ssn := ssnToCopy.Copy()
	defer ssn.Close()

	//create type for communicating rita-bl results
	resultsChannel := make(resultsChan)

	go checkRitaBlacklistIPs(destIPs, blHandle, bufferSize, resultsChannel)

	for results := range resultsChannel {
		for ipAddr, individualResults := range results {
			if len(individualResults) > 0 {
				_ = ipAddr
				//store the results
			}
		}
	}
}

func checkRitaBlacklistIPs(ips *mgo.Iter, blHandle *bl.Blacklist,
	bufferSize int, resultsChannel resultsChan) {
	i := 0
	//read in bufferSize entries and check them. Then ship them off to the writer/
	var buff = make([]ipAggregateResult, bufferSize)
	for ips.Next(&buff[i]) {
		if i == bufferSize {
			//excuse the memory hacking to get better performance
			//We need the buffer to be of type ipAggregateResult for
			//proper marshalling, but we need strings for rita-blacklist.
			//The underlying memory for ipAggregateResult is that of a string
			//since it is the only field in the struct.
			//So we can safely view buff as an array of strings using a
			//reinterpret cast. Then, we can dereference the pointer to the array
			//and use the variadic syntax to pass the array to CheckEntries.
			indexesArray := (*[]string)(unsafe.Pointer(&buff))
			resultsChannel <- blHandle.CheckEntries(list.BlacklistedIPType, (*indexesArray)...)
			//reset the buffer
			i = 0
		}
		i++
	}
	//if there are left overs in the buffer
	if i != 0 {
		buffSlice := buff[:i]
		indexesArray := (*[]string)(unsafe.Pointer(&buffSlice))
		resultsChannel <- blHandle.CheckEntries(list.BlacklistedIPType, (*indexesArray)...)
	}
	close(resultsChannel)
}
