package database

import (
	"github.com/bglebrun/rita/config"
	"os"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type (

	// MetaDBHandle exports control for the meta database
	MetaDBHandle struct {
		DB      string            // Database path
		Session *mgo.Session      // Session to the database
		lock    *sync.Mutex       // Read and write lock
		log     *log.Logger       // Logging object
		conf    *config.Resources // Keep resources object
	}

	// PFile retains everything we need to know about a given file
	PFile struct {
		ID       bson.ObjectId `bson:"_id,omitempty"`
		Path     string        `bson:"filepath"`
		Hash     string        `bson:"hash"`
		Length   int64         `bson:"length"`
		Parsed   int64         `bson:"time_complete"`
		Mod      time.Time     `bson:"modified"`
		DataBase string        `bson:"database"`
	}

	// DBMetaInfo defines some information about the database
	DBMetaInfo struct {
		ID       bson.ObjectId `bson:"_id,omitempty"` // Ident
		Name     string        `bson:"name"`          // Top level name of the database
		Analysed bool          `bson:"analyzed"`      // Has this database been analyzed
	}
)

// NewMetaDBHandle takes in a configuration and returns a MetaDBHandle controller
func NewMetaDBHandle(cfg *config.Resources) *MetaDBHandle {
	m := &MetaDBHandle{
		DB:      cfg.System.BroConfig.MetaDB,
		Session: cfg.Session.Copy(),
		log:     cfg.Log,
		lock:    new(sync.Mutex),
		conf:    cfg,
	}

	if !m.isBuilt() {
		m.newMetaDBHandle()
	}

	m.logDebug("NewMetaDBHandle", "exiting")
	return m
}

// isBuilt checks to see if a file table exists, as the existence of parsed files is prerequisite
// to the existance of anything else.
func (m *MetaDBHandle) isBuilt() bool {
	m.logDebug("isBuilt", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.Session.Copy()
	defer ssn.Close()

	coll, err := ssn.DB(m.DB).CollectionNames()
	if err != nil {
		m.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("error when looking up metadata collections")
		return false
	}

	for _, name := range coll {
		if name == "files" {
			return true
		}
	}

	m.logDebug("isBuilt", "exiting")
	return false

}

// logDebug will simply output some state info
func (m *MetaDBHandle) logDebug(function, message string) {
	m.log.WithFields(log.Fields{
		"function": function,
		"package":  "database",
		"module":   "meta",
	}).Debug(message)
}

// AddNewDB adds a new database tot he DBMetaInfo table
func (m *MetaDBHandle) AddNewDB(name string) error {
	m.logDebug("AddNewDB", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.Session.Copy()
	defer ssn.Close()

	err := ssn.DB(m.DB).C("databases").Insert(DBMetaInfo{Name: name, Analysed: false})
	if err != nil {
		m.log.WithFields(log.Fields{
			"error": err.Error(),
			"name":  name,
		}).Error("failed to create new db document")
		return err
	}

	// Do some initialization before parsing into the database
	// we have to use modconf here because database objects are initialized on a config
	// and expect only one database... no we're into the realm of tracking idividual
	// configurations around ... Bleh.
	modconf := m.conf
	modconf.System.DB = name
	d := NewDB(modconf)
	d.BuildConnectionsCollection()
	d.BuildHttpCollection()
	m.logDebug("AddNewDB", "exiting")
	return nil

}

// MarkDBCompleted marks a database as having been analyzed
func (m *MetaDBHandle) MarkDBCompleted(name string) error {
	m.logDebug("MarkDBCompleted", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.Session.Copy()
	defer ssn.Close()

	dbr := DBMetaInfo{}
	err := ssn.DB(m.DB).C("databases").
		Find(bson.M{"name": name}).One(&dbr)

	if err != nil {
		m.log.WithFields(log.Fields{
			"database_requested": name,
			"error":              err.Error(),
		}).Error("database not found in metadata directory")
		return err
	}

	err = ssn.DB(m.DB).C("databases").
		Update(bson.M{"_id": dbr.ID}, bson.M{"$set": bson.M{"analyzed": true}})

	if err != nil {
		m.log.WithFields(log.Fields{
			"metadb_attempted":   m.DB,
			"database_requested": name,
			"_id":                dbr.ID.Hex,
			"error":              err.Error(),
		}).Error("could not update database entry in meta")
		return err
	}
	m.logDebug("MarkDBCompleted", "exiting")
	return nil

}

// GetUnAnalyzedDatabases builds a list of database names which have yet to be analyzed and returns
func (m *MetaDBHandle) GetUnAnalysedDatabases() []string {

	m.logDebug("GetUnAnalysedDatabases", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.Session.Copy()
	defer ssn.Close()

	var res []string
	var cur DBMetaInfo
	iter := ssn.DB(m.DB).C("databases").Find(nil).Iter()
	for iter.Next(&cur) {
		if !cur.Analysed {
			res = append(res, cur.Name)
		}
	}
	m.logDebug("GetUnAnalysedDatabases", "exiting")
	return res

}

// GetDatabases returns a list of databases being tracked in metadb or an empty array on failure
func (m *MetaDBHandle) GetDatabases() []string {
	m.logDebug("GetDatabases", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()

	ssn := m.Session.Copy()
	defer ssn.Close()

	iter := ssn.DB(m.DB).C("databases").Find(nil).Iter()

	var res []string
	var db DBMetaInfo
	for iter.Next(&db) {
		res = append(res, db.Name)
	}
	m.logDebug("GetDatabases", "exiting")
	return res
}

// GetFiles gets a list of all PFile objects in the database if successful return a list of files
// from the database, in the case of failure return a zero length list of files and generat a log
// message.
func (m *MetaDBHandle) GetFiles() []*PFile {
	m.logDebug("GetFiles", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()
	var ret []*PFile

	ssn := m.Session.Copy()
	defer ssn.Close()

	var f PFile
	iter := ssn.DB(m.DB).C("files").Find(nil).Iter()

	for iter.Next(&f) {
		ret = append(ret, &f)
	}
	m.logDebug("GetFiles", "exiting")
	return ret
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
		m.log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("newMetaDBHandle failed to build database (aborting)")
		os.Exit(-1)
	}

	ssn := m.Session.Copy()
	defer ssn.Close()

	// Create the files collection
	myCol := mgo.CollectionInfo{
		DisableIdIndex: false,
		Capped:         false,
	}

	err := ssn.DB(m.DB).C("files").Create(&myCol)
	errchk(err)

	idx := mgo.Index{
		Key:        []string{"hash", "database"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Name:       "hashindex",
	}

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

// markComplete will mark a file as having been completed in the database
func (m *MetaDBHandle) MarkCompleted(f *PFile) error {
	m.logDebug("MarkCompleted", "entering")
	m.lock.Lock()
	defer m.lock.Unlock()
	ssn := m.Session.Copy()
	defer ssn.Close()

	pfile := PFile{}
	err := ssn.DB(m.DB).C("files").
		Find(bson.M{"hash": f.Hash, "database": f.DataBase}).One(&pfile)

	if err != nil {
		m.log.WithFields(log.Fields{
			"file":  f.Path,
			"error": err.Error(),
		}).Error("file could not be looked up by hash")
		return err
	}

	err = ssn.DB(m.DB).C("files").
		Update(bson.M{"_id": pfile.ID}, bson.M{"$set": bson.M{"time_complete": time.Now().Unix()}})

	if err != nil {
		m.log.WithFields(log.Fields{
			"file":  f.Path,
			"error": err.Error(),
		}).Error("could not update file in meta")
		return err
	}

	m.logDebug("MarkCompleted", "exiting")
	return nil

}

// updateFiles updates the files table with all of the new files from a recent walk of the dir structure
// at the end of the update we return a new GetFiles array so that the parser knows which files to get
// to parsing.
func (m *MetaDBHandle) UpdateFiles(files []*PFile) []*PFile {
	m.logDebug("UpdateFiles", "entering")
	m.lock.Lock()
	m.lock.Unlock()

	ssn := m.Session.Copy()
	defer ssn.Close()

	var work []*PFile
	myFiles := m.GetFiles()

	for _, infile := range files {
		have := false
		for _, file := range myFiles {
			if file.Hash == infile.Hash {
				have = true
				if file.Parsed > 0 && infile.Parsed > file.Parsed {
					m.log.WithFields(log.Fields{
						"warn": "mismatched parse times",
						"path": file.Path,
					}).Warning("file may have been parsed twice")

				}
			}
		}
		if !have {
			work = append(work, infile)
		}
	}

	cur := ssn.DB(m.DB).C("files")

	for _, f := range work {
		err := cur.Insert(f)
		if err != nil {
			m.log.WithFields(log.Fields{
				"error": err.Error(),
				"file":  f.Path,
			}).Error("Failed to insert")
		}
	}
	m.logDebug("UpdateFiles", "exiting")
	return work
}
