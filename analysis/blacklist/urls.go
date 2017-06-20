package blacklist

import (
	"strings"

	"github.com/ocmdev/rita-blacklist2/list"

	bl "github.com/ocmdev/rita-blacklist2"
	mgo "gopkg.in/mgo.v2"
)

type urlShort struct {
	URL string `bson:"url"`
	URI string `bson:"uri"`
}

func buildBlacklistedURLs(urls *mgo.Iter, ssnToCopy *mgo.Session,
	blHandle *bl.Blacklist, destCollection string, bufferSize int) {
	//create session to write to
	ssn := ssnToCopy.Copy()
	defer ssn.Close()

	//create type for communicating rita-bl results
	resultsChannel := make(resultsChan)

	go checkRitaBlacklistURLs(urls, blHandle, bufferSize, resultsChannel)

	for results := range resultsChannel {
		for url, individualResults := range results {
			if len(individualResults) > 0 {
				_ = url
				//TODO: resplit proto, url, and uri out :(
				//store the results
			}
		}
	}
}

func checkRitaBlacklistURLs(urls *mgo.Iter, blHandle *bl.Blacklist,
	bufferSize int, resultsChannel resultsChan) {
	i := 0
	//read in bufferSize entries and check them. Then ship them off to the writer/
	var buff = make([]string, bufferSize)
	var holder urlShort
	for urls.Next(&holder) {

		//assume http url if not specified
		if !strings.Contains(holder.URI, "://") {
			holder.URI = "http://" + holder.URI
		}
		buff[i] = holder.URI + holder.URL

		if i == bufferSize {
			resultsChannel <- blHandle.CheckEntries(list.BlacklistedURLType, buff...)
			//reset the buffer
			i = 0
		}
		i++
	}
	//if there are left overs in the buffer
	if i != 0 {
		resultsChannel <- blHandle.CheckEntries(list.BlacklistedURLType, buff[:i]...)
	}
	close(resultsChannel)
}
