package blacklist

import (
	"errors"
	"strings"
	"fmt"

	"github.com/ocmdev/rita-bl/list"

	bl "github.com/ocmdev/rita-bl"
	"github.com/ocmdev/rita/database"
	data "github.com/ocmdev/rita/datatypes/blacklist"
	"github.com/ocmdev/rita/datatypes/structure"
	"github.com/ocmdev/rita/datatypes/urls"
	log "github.com/sirupsen/logrus"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type urlShort struct {
	URL string `bson:"url"`
	URI string `bson:"uri"`
}

//buildBlacklistedURLs builds a set of blacklsited urls from the
//iterator provided, the system config, a handle to rita-blacklist,
//a buffer of urls to check at a time, and protocol prefix string to
//append to results coming from the iterator
func buildBlacklistedURLs(urls *mgo.Iter, res *database.Resources,
	blHandle *bl.Blacklist, bufferSize int, prefix string) {
	//create session to write to
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	outputCollection := ssn.DB(res.DB.GetSelectedDB()).C(
		res.Config.T.Blacklisted.UrlsTable,
	)
	//create type for communicating rita-bl results
	resultsChannel := make(resultsChan)

	go checkRitaBlacklistURLs(urls, blHandle, bufferSize, resultsChannel, prefix)

	//results are maps from ip addresses to arrays of their respective results
	for results := range resultsChannel {
		//loop over the map
		for url, individualResults := range results {
			//if the ip address has blacklist results
			if len(individualResults) > 0 {
				blURL := data.BlacklistedURL{}
				for _, result := range individualResults {
					blURL.Lists = append(blURL.Lists, result.List)
				}
				err := fillBlacklistedURL(
					&blURL,
					url,
					res.DB.GetSelectedDB(),
					res.Config.T.Urls.UrlsTable,
					res.Config.T.Structure.UniqueConnTable,
					res.Config.T.Structure.HTTPTable,
					ssn,
					prefix,
				)
				if err != nil {
					res.Log.WithFields(log.Fields{
						"err": err.Error(),
						"url": url,
						"db":  res.DB.GetSelectedDB(),
					}).Error("could not aggregate info on blacklisted url")
					continue
				}
				outputCollection.Insert(&blURL)
			}
		}
	}
}

func checkRitaBlacklistURLs(urls *mgo.Iter, blHandle *bl.Blacklist,
	bufferSize int, resultsChannel resultsChan, prefix string) {
	i := 0
	//read in bufferSize entries and check them. Then ship them off to the writer/
	var buff = make([]string, bufferSize)
	var holder urlShort
	for urls.Next(&holder) {
		//assume http url
		buff[i] = prefix + holder.URL + holder.URI

		if i == bufferSize-1 {
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

func fillBlacklistedURL(blURL *data.BlacklistedURL, longURL, db,
	urlCollection, uconnCollection string, httpCollection string,
	ssn *mgo.Session, prefix string) error {
	var urlQuery bson.M
	urlTrimmed := strings.TrimPrefix(longURL, prefix)
	resourceIdx := strings.Index(urlTrimmed, "/")
	if resourceIdx == -1 {
		return errors.New("url does not specify a resource")
	}
	host := urlTrimmed[:resourceIdx]
	resource := urlTrimmed[resourceIdx:]

	urlQuery = bson.M{"url": host, "uri": resource}
	var blURLFull urls.URL
	err := ssn.DB(db).C(urlCollection).Find(urlQuery).One(&blURLFull)
	if err != nil {
		return err
	}
	blURL.Host = host
	blURL.Resource = resource
	httpPipeline := []bson.D{
		{
			{ "$match", bson.M{"host": host, "uri": resource} },
		},
		{
			{ "$group", bson.M{
					"_id": bson.D{
						{ "h", "$host" },
						{ "u", "$uri" },
						{ "s", "$id_origin_h" },
						{ "d", "$id_resp_h" },
					},
					"host": bson.M{ "$first": "$host" },
					"uri": bson.M{ "$first": "$uri" },
					"src": bson.M{ "$first": "$id_origin_h" },
					"dst": bson.M{ "$first": "$id_resp_h" },
			}},
		},
		{
			{ "$lookup", bson.M{
					"from": uconnCollection,
					"localField": "dst",
					"foreignField": "dst",
					"as": "uconn",
			}},
		},
		{
			{ "$unwind", "$uconn" },
		},
		{
			{ "$redact", bson.M{
					"$cond": bson.M{
						"if": bson.M{
							"$eq": []interface{}{
								"$src",
								"$uconn.src",
							},
						},
						"then": "$$KEEP",
						"else": "$$PRUNE",
					},
			}},
		},
		{
			{ "$project", bson.M{
					"id": 1,
					"connection_count": "$uconn.connection_count",
					"src": 1,
					"dst": 1,
					"local_src": "$uconn.local_src",
					"local_dst": "$uconn.local_dst",
					"total_bytes": "$uconn.total_bytes",
					"average_bytes": "$uconn.average_bytes",
					"total_duration": "$uconn.total_duration",
			}},
		},
	}

	var totalBytes int
	var totalConnections int
	var uniqueConnCount int
	uniqueConnections := ssn.DB(db).C(httpCollection).Pipe(httpPipeline).Iter()
	var uconn structure.UniqueConnection
	for uniqueConnections.Next(&uconn) {
		totalBytes += uconn.TotalBytes
		totalConnections += uconn.ConnectionCount
		uniqueConnCount++
	}
	if uniqueConnections.Err() != nil {
		fmt.Println(uniqueConnections.Err())
	}
	blURL.Connections = totalConnections
	blURL.UniqueConnections = uniqueConnCount
	blURL.TotalBytes = totalBytes

	return nil
}
