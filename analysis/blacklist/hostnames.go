package blacklist

import (
	"unsafe"

	"github.com/activecm/rita-bl/list"
	"github.com/activecm/rita/database"
	"github.com/activecm/rita/datatypes/dns"
	"github.com/activecm/rita/datatypes/structure"

	bl "github.com/activecm/rita-bl"
	data "github.com/activecm/rita/datatypes/blacklist"
	log "github.com/sirupsen/logrus"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type hostnameShort struct {
	Host string `bson:"host"`
}

//buildBlacklistedHostnames builds a set of blacklisted hostnames from the
//iterator provided, the system config, a handle to rita-blacklist,
//and a buffer of hostnames to check at a time
func buildBlacklistedHostnames(hostnames *mgo.Iter, res *database.Resources,
	blHandle *bl.Blacklist, bufferSize int) {
	//create session to write to
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	outputCollection := ssn.DB(res.DB.GetSelectedDB()).C(
		res.Config.T.Blacklisted.HostnamesTable,
	)
	//create type for communicating rita-bl results
	resultsChannel := make(resultsChan)

	go checkRitaBlacklistHostnames(hostnames, blHandle, bufferSize, resultsChannel)

	//results are maps from ip addresses to arrays of their respective results
	for results := range resultsChannel {
		//loop over the map
		for hostname, individualResults := range results {
			//if the hostname has blacklist results
			if len(individualResults) > 0 {
				blHostname := data.BlacklistedHostname{Hostname: hostname}
				for _, result := range individualResults {
					blHostname.Lists = append(blHostname.Lists, result.List)
				}
				err := fillBlacklistedHostname(
					&blHostname,
					res.DB.GetSelectedDB(),
					res.Config.T.DNS.HostnamesTable,
					res.Config.T.Structure.UniqueConnTable,
					ssn,
				)
				if err != nil {
					res.Log.WithFields(log.Fields{
						"err":      err.Error(),
						"hostname": hostname,
						"db":       res.DB.GetSelectedDB(),
					}).Error("could not aggregate info on blacklisted hostname")
					continue
				}
				outputCollection.Insert(&blHostname)
			}
		}
	}
}

func checkRitaBlacklistHostnames(hostnames *mgo.Iter, blHandle *bl.Blacklist,
	bufferSize int, resultsChannel resultsChan) {
	i := 0
	//read in bufferSize entries and check them. Then ship them off to the writer
	var buff = make([]hostnameShort, bufferSize)
	for hostnames.Next(&buff[i]) {
		if i == bufferSize-1 {
			//see comment in checkRitaBlacklistIPs
			indexesArray := (*[]string)(unsafe.Pointer(&buff))
			resultsChannel <- blHandle.CheckEntries(
				list.BlacklistedHostnameType, (*indexesArray)...,
			)
			//reset the buffer
			i = 0
		}
		i++
	}
	//if there are left overs in the buffer
	if i != 0 {
		buffSlice := buff[:i]
		indexesArray := (*[]string)(unsafe.Pointer(&buffSlice))
		resultsChannel <- blHandle.CheckEntries(
			list.BlacklistedHostnameType, (*indexesArray)...,
		)
	}
	close(resultsChannel)
}

func fillBlacklistedHostname(blHostname *data.BlacklistedHostname, db,
	hostnamesCollection, uconnCollection string, ssn *mgo.Session) error {
	hostnameQuery := bson.M{"host": blHostname.Hostname}
	var blHostnameFull dns.Hostname
	err := ssn.DB(db).C(hostnamesCollection).Find(hostnameQuery).One(&blHostnameFull)
	if err != nil {
		return err
	}

	connQuery := bson.M{"dst": bson.M{"$in": blHostnameFull.IPs}}

	var totalBytes int
	var totalConnections int
	var uniqueConnCount int
	uniqueConnections := ssn.DB(db).C(uconnCollection).Find(connQuery).Iter()
	var uconn structure.UniqueConnection
	for uniqueConnections.Next(&uconn) {
		totalBytes += uconn.TotalBytes
		totalConnections += uconn.ConnectionCount
		uniqueConnCount++
	}
	blHostname.Connections = totalConnections
	blHostname.UniqueConnections = uniqueConnCount
	blHostname.TotalBytes = totalBytes

	return nil
}
