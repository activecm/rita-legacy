package sanitization

import (
	"fmt"
	"net/url"
	"strings"

	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"github.com/activecm/rita/parser/parsetypes"
	"github.com/activecm/rita/resources"
)

//SanitizeData cleans up abnormalities in the imported data
func SanitizeData(res *resources.Resources) {
	sanitizeHTTPData(res)
}

//sanitizeHTTPData cleans up abnormalities in the HTTP collection
func sanitizeHTTPData(res *resources.Resources) {
	sess := res.DB.Session.Copy()
	defer sess.Close()

	http := sess.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.HTTPTable)

	var httpRec parsetypes.HTTP
	httpIter := http.Find(nil).Iter()

	bufferSize := res.Config.S.Bro.ImportBuffer
	if bufferSize%2 == 1 {
		bufferSize++
	}

	buffer := make([]interface{}, 0, bufferSize)

	for httpIter.Next(&httpRec) {
		updateDoc := sanitizeHTTPRecord(&httpRec)

		if updateDoc == nil {
			continue
		}

		if len(buffer) == bufferSize {
			err := commitUpdateBuffer(buffer, http)
			if err != nil {
				res.Log.Error("Could not sanitize http records", err)
				return
			}
			buffer = buffer[:0]
		}
		buffer = append(buffer, bson.M{"_id": httpRec.ID})
		buffer = append(buffer, updateDoc)
	}

	if len(buffer) > 0 {
		err := commitUpdateBuffer(buffer, http)
		if err != nil {
			res.Log.Error("Could not sanitize http records", err)
			return
		}
	}
}

func sanitizeHTTPRecord(httpRec *parsetypes.HTTP) bson.M {
	newURI := httpRec.URI

	// ex: Host: 67.217.65.244 URI: 67.217.65.244:443
	// URI -> :443 which will cause an error in the parser
	if strings.HasPrefix(httpRec.URI, httpRec.Host) {
		newURI = httpRec.URI[len(httpRec.Host):]
	}

	parsedURL, err := url.Parse(newURI)
	if err != nil {
		newURI = ""
	}

	//CASE: Host: www.google.com URI: http://www.google.com
	if err == nil && parsedURL.IsAbs() {
		newURI = parsedURL.RequestURI()
	}

	fmt.Println(newURI, httpRec.URI)

	if newURI == httpRec.URI {
		return nil
	}

	//nolint: vet
	return bson.M{
		"$set": bson.D{
			{"uri", newURI},
		},
	}
}

func commitUpdateBuffer(buffer []interface{}, collection *mgo.Collection) error {
	bulk := collection.Bulk()
	bulk.Unordered()
	bulk.Update(buffer...)
	_, err := bulk.Run()
	return err
}
