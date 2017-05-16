package database

import (
	"os"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	fpt "github.com/ocmdev/rita/parser/fileparsetypes"
	"github.com/rifflock/lfshook"
	"github.com/weekface/mgorus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type (

	// MetaDBHandle exports control for the meta database
	MetaDBHandle struct {
		DB   string      // Database path
		lock *sync.Mutex // Read and write lock
		log  *log.Logger // Logging object
		res  *Resources  // Keep resources object
	}

	// DBMetaInfo defines some information about the database
	DBMetaInfo struct {
		ID         bson.ObjectId `bson:"_id,omitempty"` // Ident
		Name       string        `bson:"name"`          // Top level name of the database
		Analyzed   bool          `bson:"analyzed"`      // Has this database been analyzed
		UsingDates bool          `bson:"dates"`         // Whether this db was created with dates enabled
		Version    string        `bson:"version"`       // Rita version at import
	}
)

// AddNewDB adds a new database to the DBMetaInfo table
func (m *MetaDBHandle) AddNewDB(name string) error {
	m.logDebug("AddNewDB", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()

	err := ssn.DB(m.DB).C("databases").Insert(
		DBMetaInfo{
			Name:       name,
			Analyzed:   false,
			UsingDates: m.res.System.BroConfig.UseDates,
			Version:    m.res.System.Version,
		},
	)
	if err != nil {
		m.res.Log.WithFields(log.Fields{
			"error": err.Error(),
			"name":  name,
		}).Error("failed to create new db document")
		return err
	}

	m.logDebug("AddNewDB", "exiting")
	return nil
}

// DeleteDB removes a database managed by RITA
func (m *MetaDBHandle) DeleteDB(name string) error {
	m.logDebug("DeleteDB", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()

	//get the record
	var db DBMetaInfo
	err := ssn.DB(m.DB).C("databases").Find(bson.M{"name": name}).One(&db)
	if err != nil {
		return err
	}

	//delete the record
	err = ssn.DB(m.DB).C("databases").Remove(bson.M{"name": name})
	if err != nil {
		return err
	}

	//drop the data
	ssn.DB(name).DropDatabase()

	//delete any parsed file records associated
	_, err = ssn.DB(m.DB).C("files").RemoveAll(bson.M{"database": name})
	if err != nil {
		return err
	}

	m.logDebug("DeleteDB", "exiting")
	return nil
}

// MarkDBAnalyzed marks a database as having been analyzed
func (m *MetaDBHandle) MarkDBAnalyzed(name string, complete bool) error {
	m.logDebug("MarkDBAnalyzed", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()

	dbr := DBMetaInfo{}
	err := ssn.DB(m.DB).C("databases").
		Find(bson.M{"name": name}).One(&dbr)

	if err != nil {
		m.res.Log.WithFields(log.Fields{
			"database_requested": name,
			"error":              err.Error(),
		}).Error("database not found in metadata directory")
		return err
	}

	err = ssn.DB(m.DB).C("databases").
		Update(bson.M{"_id": dbr.ID}, bson.M{"$set": bson.M{"analyzed": complete}})

	if err != nil {
		m.res.Log.WithFields(log.Fields{
			"metadb_attempted":   m.DB,
			"database_requested": name,
			"_id":                dbr.ID.Hex,
			"error":              err.Error(),
		}).Error("could not update database entry in meta")
		return err
	}
	m.logDebug("MarkDBAnalyzed", "exiting")
	return nil
}

// GetDBMetaInfo returns a meta db entry
func (m *MetaDBHandle) GetDBMetaInfo(name string) (DBMetaInfo, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()
	var result DBMetaInfo
	err := ssn.DB(m.DB).C("databases").Find(bson.M{"name": name}).One(&result)
	return result, err
}

// GetDatabases returns a list of databases being tracked in metadb or an empty array on failure
func (m *MetaDBHandle) GetDatabases() []string {
	m.logDebug("GetDatabases", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()

	iter := ssn.DB(m.DB).C("databases").Find(nil).Iter()

	var results []string
	var db DBMetaInfo
	for iter.Next(&db) {
		results = append(results, db.Name)
	}
	m.logDebug("GetDatabases", "exiting")
	return results
}

// GetUnAnalyzedDatabases builds a list of database names which have yet to be analyzed
func (m *MetaDBHandle) GetUnAnalyzedDatabases() []string {
	m.logDebug("GetUnAnalyzedDatabases", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()

	var results []string
	var cur DBMetaInfo
	iter := ssn.DB(m.DB).C("databases").Find(bson.M{"analyzed": false}).Iter()
	for iter.Next(&cur) {
		results = append(results, cur.Name)
	}
	m.logDebug("GetUnAnalyzedDatabases", "exiting")
	return results
}

// GetAnalyzedDatabases builds a list of database names which have been analyzed
func (m *MetaDBHandle) GetAnalyzedDatabases() []string {
	m.logDebug("GetUnAnalyzedDatabases", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()

	var results []string
	var cur DBMetaInfo
	iter := ssn.DB(m.DB).C("databases").Find(bson.M{"analyzed": true}).Iter()
	for iter.Next(&cur) {
		results = append(results, cur.Name)
	}
	m.logDebug("GetUnAnalyzedDatabases", "exiting")
	return results
}

///////////////////////////////////////////////////////////////////////////////
//                            File Processing                                //
///////////////////////////////////////////////////////////////////////////////

// GetFiles gets a list of all IndexedFile objects in the database if successful return a list of files
// from the database, in the case of failure return a zero length list of files and generat a log
// message.
func (m *MetaDBHandle) GetFiles() ([]fpt.IndexedFile, error) {
	m.logDebug("GetFiles", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()
	var toReturn []fpt.IndexedFile

	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()

	//TODO: Make this read the database from the config file
	err := ssn.DB(m.DB).C("files").Find(nil).Iter().All(&toReturn)
	if err != nil {
		m.res.Log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("could not fetch files from meta database")
		return nil, err
	}
	m.logDebug("GetFiles", "exiting")
	return toReturn, nil
}

//AddParsedFiles adds indexed files to the files the metaDB using the bulk API
func (m *MetaDBHandle) AddParsedFiles(files []*fpt.IndexedFile) error {
	m.logDebug("AddParsedFiles", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()

	//TODO: Make this read the database from the config file
	bulk := ssn.DB(m.DB).C("files").Bulk()
	bulk.Unordered()

	//construct the interface slice for bulk
	interfaceSlice := make([]interface{}, len(files))
	for i, d := range files {
		interfaceSlice[i] = *d
	}

	bulk.Insert(interfaceSlice...)
	_, err := bulk.Run()
	if err != nil {
		m.res.Log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("could not insert files into meta database")
		return err
	}
	return nil
}

/////////////////////

// isBuilt checks to see if a file table exists, as the existence of parsed files is prerequisite
// to the existance of anything else.
func (m *MetaDBHandle) isBuilt() bool {
	m.logDebug("isBuilt", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()

	coll, err := ssn.DB(m.DB).CollectionNames()
	if err != nil {
		m.res.Log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("error when looking up metadata collections")
		return false
	}

	for _, name := range coll {
		//TODO: Make this read the database from the config file
		if name == "files" {
			return true
		}
	}

	m.logDebug("isBuilt", "exiting")
	return false
}

// newMetaDBHandle creates a new metadata database failure is not an option,
// if this function fails it will bring down the system.
func (m *MetaDBHandle) newMetaDBHandle() {
	m.logDebug("newMetaDBHandle", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()

	errchk := func(err error) {
		if err == nil {
			return
		}
		m.res.Log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("newMetaDBHandle failed to build database (aborting)")
		os.Exit(-1)
	}

	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()

	// Log Hooks init, spin off MGOrus hook and lfshook
	m.res.Log = log.New()

	mgohook, mgoHkErr := mgorus.NewHooker(m.res.System.DatabaseHost, m.res.System.BroConfig.MetaDB, "Logs")
	if mgoHkErr == nil {
		m.res.Log.Hooks.Add(mgohook)
	} else {
		errchk(mgoHkErr)
	}

	dir := m.res.System.RitaLogPath

	m.res.Log.Hooks.Add(lfshook.NewHook(lfshook.PathMap{
		log.InfoLevel:  dir + "/info-" + time.Now().Format("2006-01-02-15:04") + ".log",
		log.ErrorLevel: dir + "/error-" + time.Now().Format("2006-01-02-15:04") + ".log",
	}))

	// Create the files collection
	myCol := mgo.CollectionInfo{
		DisableIdIndex: false,
		Capped:         false,
	}

	//TODO: Make this read the database from the config file
	err := ssn.DB(m.DB).C("files").Create(&myCol)
	errchk(err)

	idx := mgo.Index{
		Key:        []string{"hash", "database"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Name:       "hashindex",
	}

	//TODO: Make this read the database from the config file
	err = ssn.DB(m.DB).C("files").EnsureIndex(idx)
	errchk(err)

	// Create the blacklist collection
	err = ssn.DB(m.DB).C("blacklisted").Create(&myCol)
	errchk(err)

	idx = mgo.Index{
		Key:        []string{"remote_host"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Name:       "rhost_index",
	}

	err = ssn.DB(m.DB).C("blacklisted").EnsureIndex(idx)
	errchk(err)

	// Create the database collection
	err = ssn.DB(m.DB).C("databases").Create(&myCol)
	errchk(err)

	idx = mgo.Index{
		Key:        []string{"name"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Name:       "nameindex",
	}

	err = ssn.DB(m.DB).C("databases").EnsureIndex(idx)
	errchk(err)

	m.logDebug("newMetaDBHandle", "exiting")
}

// logDebug will simply output some state info
func (m *MetaDBHandle) logDebug(function, message string) {
	m.res.Log.WithFields(log.Fields{
		"function": function,
		"package":  "database",
		"module":   "meta",
	}).Debug(message)
}
