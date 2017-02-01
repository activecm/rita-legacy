package database

import (
	"time"

	"github.com/bglebrun/rita/config"
	"github.com/weekface/mgorus"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// DB is the workhorse container for messing with the database
type DB struct {
	r *config.Resources
	l *log.Logger
	// s  *mgo.Session
	db string
}

// Hook our logger into MongoDB
func init() {
	hooker, err := mgorus.NewHooker("localhost:27017", "ritaErr", "runErr")

	if err == nil {
		log.AddHook(hooker)
	} else {
		log.WithFields(log.Fields{
			"Database Hook": "Not connected!",
		}).Warn("Log could not be hooked into MongoDB, errors will not be logged!")
	}
}

// NewDB builds up a new data session
func NewDB(cfg *config.Resources) *DB {
	d := new(DB)
	d.db = cfg.System.DB
	d.l = cfg.Log
	// d.s = cfg.Session
	d.r = cfg
	return d
}

///////////////////////////////////////////////////////////////////////////////
////////////////////////// SUPPORTING FUNCTIONS ///////////////////////////////
///////////////////////////////////////////////////////////////////////////////

/*
 * Name:     collectionExists
 * Purpose:  Returns true if collection exists in database
 * comments:
 */
func (d *DB) collectionExists(table string) bool {
	ssn := d.r.Session.Copy()
	defer ssn.Close()
	coll, err := ssn.DB(d.db).CollectionNames()
	if err != nil {
		d.l.WithFields(log.Fields{
			"error": err.Error(),
		}).Panic("Failed collection name lookup")
		panic("Failed collection lookup")
	}
	for _, name := range coll {
		if name == table {
			return true
		}
	}
	return false

}

/*
 * Name:     createNewCollection
 * Purpose:  Creates a new collection with required indeces
 * comments:
 */
func (d *DB) createCollection(name string, indeces []string) string {
	// Make a copy of the current session
	session := d.r.Session.Copy()
	defer session.Close()

	if len(name) < 1 {
		d.l.Debug("Error, check the collection name in yaml file and systemConfig: ", name)
		return " (Name error: check collection name in yaml file and config) "
	}

	// Check if ollection already exists
	if d.collectionExists(name) {
		d.l.Debug("Collection already exists:", name)
		return " (Collection already exists!) "
	}

	d.l.Info("Building collection: ", name)

	// Create new collection by referencing to it, no need to call Create
	collection := session.DB(d.r.System.DB).C(name)

	// Make sure it actually got created
	if d.collectionExists(name) {
		d.l.Debug("Error, check the collection name in yaml file and systemConfig: ", name)
		return " (Name error: check the collection name in yaml file and config) "
	}

	for _, val := range indeces {
		index := mgo.Index{
			Key: []string{val},
		}
		err := collection.EnsureIndex(index)
		if err != nil {
			d.l.WithFields(log.Fields{
				"error": err.Error(),
			}).Panic("Failed to create indeces")
			return " (Failed to create indeces!) "
		}
	}

	return ""
}

/*
 * Name:     aggregateCollection
 * Purpose:  Builds collections that are built via aggregation
 * comments:
 */
func (d *DB) aggregateCollection(source_collection_name string, pipeline []bson.D, results interface{}) {
	// Make a copy of the current session
	session := d.r.Session.Copy()
	defer session.Close()

	session.SetSocketTimeout(2 * time.Hour)

	// Setup a container for the results
	// results := []bson.D{}

	// Identify the source collection we will aggregate information from into the new collection
	if !d.collectionExists(source_collection_name) {
		d.l.Info("Failed aggregation: (Source collection: ", source_collection_name, " doesn't exist)")
		return //results
	}
	source_collection := session.DB(d.r.System.DB).C(source_collection_name)

	// Create the pipe
	pipe := source_collection.Pipe(pipeline).AllowDiskUse()

	err := pipe.All(results)

	// If error, Throw computer against wall and drink 2 angry beers while
	// questioning your life, purpose, and relationships.
	if err != nil {
		d.l.WithFields(log.Fields{
			"error": err.Error(),
		}).Panic("Failed aggregate operation")
		return
	}

}

/*
 * Name:     mapReduce Collection
 * Purpose:  Builds collections that are built via map reduce
 * comments:
 */
func (d *DB) mapReduceCollection(source_collection_name string, job mgo.MapReduce) bool {
	// Make a copy of the current session
	session := d.r.Session.Copy()
	defer session.Close()

	session.SetSocketTimeout(2 * time.Hour)

	// Identify the source collection we will aggregate information from into the new collection
	if !d.collectionExists(source_collection_name) {
		d.l.Info("Failed map reduce: (Source collection: ", source_collection_name, " doesn't exist)")
		return false
	}
	source_collection := session.DB(d.r.System.DB).C(source_collection_name)

	// Map reduce that shit
	_, err := source_collection.Find(nil).MapReduce(&job, nil)

	// If error, Throw computer against wall and drink 2 angry beers while
	// questioning your life, purpose, and relationships.
	if err != nil {
		d.l.Error("Failed map reduce for: ", source_collection_name, err)
		return false
	}

	return true
}
