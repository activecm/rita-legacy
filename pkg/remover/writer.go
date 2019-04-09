package remover

import (
	"fmt"
	"sync"

	"github.com/activecm/rita/config"
	"github.com/activecm/rita/database"
	"github.com/globalsign/mgo/bson"
)

type (
	writer struct {
		cid               int            // chuck id for deletion
		db                *database.DB   // provides access to MongoDB
		conf              *config.Config // contains details needed to access MongoDB
		cidRemoverChannel chan string    // holds target collection names
		updaterChannel    chan update    // holds update queries
		writeWg           sync.WaitGroup // wait for writing to finish
	}
)

//newCIDRemover creates a new writer object to write output data
func newCIDRemover(cid int, db *database.DB, conf *config.Config) *writer {
	return &writer{
		cid:               cid,
		db:                db,
		conf:              conf,
		cidRemoverChannel: make(chan string),
	}
}

//newUpdater creates a new writer object to write output data
func newUpdater(cid int, db *database.DB, conf *config.Config) *writer {
	return &writer{
		cid:            cid,
		db:             db,
		conf:           conf,
		updaterChannel: make(chan update),
	}
}

//collect sends a group of results to the writer for writing out to the database
func (w *writer) collectCIDRemover(data string) {
	w.cidRemoverChannel <- data
}

//collect sends a group of results to the writer for writing out to the database
func (w *writer) collectUpdater(data update) {
	w.updaterChannel <- data
}

//closeCIDRemover waits for the write threads to finish
func (w *writer) closeCIDRemover() {
	close(w.cidRemoverChannel)
	w.writeWg.Wait()
}

//closeUpdater waits for the write threads to finish
func (w *writer) closeUpdater() {
	close(w.updaterChannel)
	w.writeWg.Wait()
}

//startCIDRemover kicks off a new write thread
func (w *writer) startCIDRemover() {
	w.writeWg.Add(1)
	go func() {
		ssn := w.db.Session.Copy()
		defer ssn.Close()

		for data := range w.cidRemoverChannel {

			//delete the ENTIRE record if it hasn't been updated since the chunk we are trying to remove
			info, err := ssn.DB(w.db.GetSelectedDB()).C(data).RemoveAll(bson.M{"cid": w.cid})
			if err != nil ||
				((info.Updated == 0) && (info.Removed == 0) && (info.Matched != 0)) {
				fmt.Println("failed to delete whole document: ", err, info, data)
			}

			// this ONLY deletes a specific chunk's DATA from a record that HAS been updated recently and doesn't need to be completely
			// removed - only the target chunk's stats should be removed from it
			info, err = ssn.DB(w.db.GetSelectedDB()).C(data).UpdateAll(bson.M{"dat.cid": w.cid}, bson.M{"$pull": bson.M{"dat": bson.M{"cid": w.cid}}})
			if err != nil ||
				((info.Updated == 0) && (info.Removed == 0) && (info.Matched != 0)) {
				fmt.Println("failed to delete chunk: ", err, info, data)
			}

		}
		w.writeWg.Done()
	}()
}

//startUpdater kicks off a new write thread
func (w *writer) startUpdater() {
	w.writeWg.Add(1)
	go func() {
		ssn := w.db.Session.Copy()
		defer ssn.Close()

		for data := range w.updaterChannel {

			err := ssn.DB(w.db.GetSelectedDB()).C(data.collection).Update(data.selector, data.query)
			if err != nil {
				fmt.Println(err, data)
			}

		}
		w.writeWg.Done()
	}()
}
