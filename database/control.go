package database

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// DB is the workhorse container for messing with the database
type DB struct {
	Session   *mgo.Session
	resources *Resources
	selected  string
}

///////////////////////////////////////////////////////////////////////////////
////////////////////////// SUPPORTING FUNCTIONS ///////////////////////////////
///////////////////////////////////////////////////////////////////////////////

func (d *DB) SelectDB(db string) {
	d.selected = db
}

func (d *DB) GetSelectedDB() string {
	return d.selected
}

/*
 * Name:     CollectionExists
 * Purpose:  Returns true if collection exists in database
 * comments:
 */
func (d *DB) CollectionExists(table string) bool {
	ssn := d.Session.Copy()
	defer ssn.Close()
	coll, err := ssn.DB(d.selected).CollectionNames()
	if err != nil {
		d.resources.Log.WithFields(log.Fields{
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
func (d *DB) CreateCollection(name string, indeces []string) string {
	// Make a copy of the current session
	session := d.Session.Copy()
	defer session.Close()

	if len(name) < 1 {
		d.resources.Log.Debug("Error, check the collection name in yaml file and systemConfig: ", name)
		return " (Name error: check collection name in yaml file and config) "
	}

	// Check if ollection already exists
	if d.CollectionExists(name) {
		d.resources.Log.Debug("Collection already exists:", name)
		return " (Collection already exists!) "
	}

	d.resources.Log.Info("Building collection: ", name)

	// Create new collection by referencing to it, no need to call Create
	collection := session.DB(d.selected).C(name)

	// Make sure it actually got created
	if d.CollectionExists(name) {
		d.resources.Log.Debug("Error, check the collection name in yaml file and systemConfig: ", name)
		return " (Name error: check the collection name in yaml file and config) "
	}

	for _, val := range indeces {
		index := mgo.Index{
			Key: []string{val},
		}
		err := collection.EnsureIndex(index)
		if err != nil {
			d.resources.Log.WithFields(log.Fields{
				"error": err.Error(),
			}).Panic("Failed to create indeces")
			return " (Failed to create indeces!) "
		}
	}

	return ""
}

/*
 * Name:     AggregateCollection
 * Purpose:  Builds collections that are built via aggregation
 * comments:
 */
func (d *DB) AggregateCollection(source_collection_name string, pipeline []bson.D) *mgo.Iter {
	// Make a copy of the current session
	session := d.Session.Copy()
	defer session.Close()

	session.SetSocketTimeout(2 * time.Hour)

	// Setup a container for the results
	// results := []bson.D{}

	// Identify the source collection we will aggregate information from into the new collection
	if !d.CollectionExists(source_collection_name) {
		d.resources.Log.Info("Failed aggregation: (Source collection: ", source_collection_name, " doesn't exist)")
		return nil
	}
	source_collection := session.DB(d.selected).C(source_collection_name)

	// Create the pipe
	pipe := source_collection.Pipe(pipeline).AllowDiskUse()

	iter := pipe.Iter()

	// If error, Throw computer against wall and drink 2 angry beers while
	// questioning your life, purpose, and relationships.
	if iter.Err() != nil {
		d.resources.Log.WithFields(log.Fields{
			"error": iter.Err().Error(),
		}).Panic("Failed aggregate operation")
		return nil
	}
	return iter
}

/*
 * Name:     mapReduce Collection
 * Purpose:  Builds collections that are built via map reduce
 * comments:
 */
func (d *DB) MapReduceCollection(source_collection_name string, job mgo.MapReduce) bool {
	// Make a copy of the current session
	session := d.Session.Copy()
	defer session.Close()

	session.SetSocketTimeout(2 * time.Hour)

	// Identify the source collection we will aggregate information from into the new collection
	if !d.CollectionExists(source_collection_name) {
		d.resources.Log.Info("Failed map reduce: (Source collection: ", source_collection_name, " doesn't exist)")
		return false
	}
	source_collection := session.DB(d.selected).C(source_collection_name)

	// Map reduce that shit
	_, err := source_collection.Find(nil).MapReduce(&job, nil)

	// If error, Throw computer against wall and drink 2 angry beers while
	// questioning your life, purpose, and relationships.
	if err != nil {
		d.resources.Log.Error("Failed map reduce for: ", source_collection_name, err)
		return false
	}

	return true
}
