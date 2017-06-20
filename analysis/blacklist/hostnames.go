package blacklist

import (
	"unsafe"

	"github.com/ocmdev/rita-blacklist2/list"

	bl "github.com/ocmdev/rita-blacklist2"
	mgo "gopkg.in/mgo.v2"
)

type hostnameShort struct {
	Host string `bson:"host"`
}

func buildBlacklistedHostnames(hostnames *mgo.Iter, ssnToCopy *mgo.Session,
	blHandle *bl.Blacklist, destCollection string, bufferSize int) {
	//create session to write to
	ssn := ssnToCopy.Copy()
	defer ssn.Close()

	//create type for communicating rita-bl results
	resultsChannel := make(resultsChan)

	go checkRitaBlacklistHostnames(hostnames, blHandle, bufferSize, resultsChannel)

	for results := range resultsChannel {
		for hostname, individualResults := range results {
			if len(individualResults) > 0 {
				_ = hostname
				//store the results
			}
		}
	}
}

func checkRitaBlacklistHostnames(hostnames *mgo.Iter, blHandle *bl.Blacklist,
	bufferSize int, resultsChannel resultsChan) {
	i := 0
	//read in bufferSize entries and check them. Then ship them off to the writer/
	var buff = make([]hostnameShort, bufferSize)
	for hostnames.Next(&buff[i]) {
		if i == bufferSize {
			//see comment in checkRitaBlacklistIPs
			indexesArray := (*[]string)(unsafe.Pointer(&buff))
			resultsChannel <- blHandle.CheckEntries(list.BlacklistedHostnameType, (*indexesArray)...)
			//reset the buffer
			i = 0
		}
		i++
	}
	//if there are left overs in the buffer
	if i != 0 {
		buffSlice := buff[:i]
		indexesArray := (*[]string)(unsafe.Pointer(&buffSlice))
		resultsChannel <- blHandle.CheckEntries(list.BlacklistedHostnameType, (*indexesArray)...)
	}
	close(resultsChannel)
}
