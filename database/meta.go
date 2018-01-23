package database

import (
	"os"
	"sync"

	fpt "github.com/ocmdev/rita/parser/fileparsetypes"
	log "github.com/sirupsen/logrus"
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
		ID             bson.ObjectId `bson:"_id,omitempty"`   // Ident
		Name           string        `bson:"name"`            // Top level name of the database
		Analyzed       bool          `bson:"analyzed"`        // Has this database been analyzed
		UsingDates     bool          `bson:"dates"`           // Whether this db was created with dates enabled
		ImportVersion  string        `bson:"import_version"`  // Rita version at import
		AnalyzeVersion string        `bson:"analyze_version"` // Rita version at analyze
	}
)

// AddNewDB adds a new database to the DBMetaInfo table
func (m *MetaDBHandle) AddNewDB(name string) error {
	m.logDebug("AddNewDB", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()

	err := ssn.DB(m.DB).C(m.res.Config.T.Meta.DatabasesTable).Insert(
		DBMetaInfo{
			Name:          name,
			Analyzed:      false,
			UsingDates:    m.res.Config.S.Bro.UseDates,
			ImportVersion: m.res.Config.S.Version,
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
	err := ssn.DB(m.DB).C(m.res.Config.T.Meta.DatabasesTable).Find(bson.M{"name": name}).One(&db)
	if err != nil {
		return err
	}

	//delete the record
	err = ssn.DB(m.DB).C(m.res.Config.T.Meta.DatabasesTable).Remove(bson.M{"name": name})
	if err != nil {
		return err
	}

	//drop the data
	ssn.DB(name).DropDatabase()

	//delete any parsed file records associated
	_, err = ssn.DB(m.DB).C(m.res.Config.T.Meta.FilesTable).RemoveAll(bson.M{"database": name})
	if err != nil {
		return err
	}
	if db.UsingDates {
		date := name[len(name)-10:]
		name = name[:len(name)-11]
		_, err = ssn.DB(m.DB).C("files").RemoveAll(
			bson.M{"database": name, "dates": date},
		)
		if err != nil {
			return err
		}
	} else {
		_, err = ssn.DB(m.DB).C("files").RemoveAll(bson.M{"database": name})
		if err != nil {
			return err
		}
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
	err := ssn.DB(m.DB).C(m.res.Config.T.Meta.DatabasesTable).
		Find(bson.M{"name": name}).One(&dbr)

	if err != nil {
		m.res.Log.WithFields(log.Fields{
			"database_requested": name,
			"error":              err.Error(),
		}).Error("database not found in metadata directory")
		return err
	}

	err = ssn.DB(m.DB).C(m.res.Config.T.Meta.DatabasesTable).
		Update(bson.M{"_id": dbr.ID}, bson.M{
			"$set": bson.D{
				{"analyzed", complete},
				{"analyze_version", m.res.Config.S.Version},
			},
		})

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
	err := ssn.DB(m.DB).C(m.res.Config.T.Meta.DatabasesTable).Find(bson.M{"name": name}).One(&result)
	return result, err
}

// GetDatabases returns a list of databases being tracked in metadb or an empty array on failure
func (m *MetaDBHandle) GetDatabases() []string {
	m.logDebug("GetDatabases", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()

	iter := ssn.DB(m.DB).C(m.res.Config.T.Meta.DatabasesTable).Find(nil).Iter()

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
	iter := ssn.DB(m.DB).C(m.res.Config.T.Meta.DatabasesTable).Find(bson.M{"analyzed": false}).Iter()
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
	iter := ssn.DB(m.DB).C(m.res.Config.T.Meta.DatabasesTable).Find(bson.M{"analyzed": true}).Iter()
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

	err := ssn.DB(m.DB).C(m.res.Config.T.Meta.FilesTable).
		Find(nil).Iter().All(&toReturn)
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
	if len(files) == 0 {
		return nil
	}
	ssn := m.res.DB.Session.Copy()
	defer ssn.Close()

	bulk := ssn.DB(m.DB).C(m.res.Config.T.Meta.FilesTable).Bulk()
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
		if name == m.res.Config.T.Meta.FilesTable {
			return true
		}
	}

	m.logDebug("isBuilt", "exiting")
	return false
}

// createMetaDB creates a new metadata database failure is not an option,
// if this function fails it will bring down the system.
func (m *MetaDBHandle) createMetaDB() {
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

	// Create the files collection
	myCol := mgo.CollectionInfo{
		DisableIdIndex: false,
		Capped:         false,
	}

	err := ssn.DB(m.DB).C(m.res.Config.T.Log.RitaLogTable).Create(&myCol)
	errchk(err)

	err = ssn.DB(m.DB).C(m.res.Config.T.Meta.FilesTable).Create(&myCol)
	errchk(err)

	idx := mgo.Index{
		Key:        []string{"hash", "database"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Name:       "hashindex",
	}

	err = ssn.DB(m.DB).C(m.res.Config.T.Meta.FilesTable).EnsureIndex(idx)
	errchk(err)

	// Create the database collection
	err = ssn.DB(m.DB).C(m.res.Config.T.Meta.DatabasesTable).Create(&myCol)
	errchk(err)

	idx = mgo.Index{
		Key:        []string{"name"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Name:       "nameindex",
	}

	err = ssn.DB(m.DB).C(m.res.Config.T.Meta.DatabasesTable).EnsureIndex(idx)
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
