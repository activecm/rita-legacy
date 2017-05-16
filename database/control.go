package database

import (
	"errors"
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

//SelectDB selects a database for analysis
func (d *DB) SelectDB(db string) {
	d.selected = db
}

//GetSelectedDB retrieves the currently selected database for analysis
func (d *DB) GetSelectedDB() string {
	return d.selected
}

//CollectionExists returns true if collection exists in the currently
//selected database
func (d *DB) CollectionExists(table string) bool {
	ssn := d.Session.Copy()
	defer ssn.Close()
	coll, err := ssn.DB(d.selected).CollectionNames()
	if err != nil {
		d.resources.Log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Failed collection name lookup")
		return false
	}
	for _, name := range coll {
		if name == table {
			return true
		}
	}
	return false
}

//CreateCollection creates a new collection in the currently selected
//database with the required indeces
func (d *DB) CreateCollection(name string, indeces []string) error {
	// Make a copy of the current session
	session := d.Session.Copy()
	defer session.Close()

	if len(name) < 1 {
		d.resources.Log.Debug("Error, check the collection name in yaml file and systemConfig: ", name)
		return errors.New("name error: check collection name in yaml file and config")
	}

	// Check if ollection already exists
	if d.CollectionExists(name) {
		d.resources.Log.Debug("Collection already exists:", name)
		return errors.New("collection already exists")
	}

	d.resources.Log.Info("Building collection: ", name)

	// Create new collection by referencing to it, no need to call Create
	err := session.DB(d.selected).C(name).Create(
		&mgo.CollectionInfo{},
	)

	// Make sure it actually got created
	if err != nil {
		d.resources.Log.Error("Error, check the collection name in yaml file and systemConfig: ", name)
		return err
	}

	collection := session.DB(d.selected).C(name)
	for _, val := range indeces {
		index := mgo.Index{
			Key: []string{val},
		}
		err := collection.EnsureIndex(index)
		if err != nil {
			d.resources.Log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Failed to create indeces")
			return err
		}
	}

	return nil
}

//AggregateCollection builds a collection via a MongoDB pipeline
func (d *DB) AggregateCollection(sourceCollection string,
	session *mgo.Session, pipeline []bson.D) *mgo.Iter {
	session.SetSocketTimeout(2 * time.Hour)

	// Identify the source collection we will aggregate information from into the new collection
	if !d.CollectionExists(sourceCollection) {
		d.resources.Log.Info("Failed aggregation: (Source collection: ",
			sourceCollection, " doesn't exist)")
		return nil
	}
	collection := session.DB(d.selected).C(sourceCollection)

	// Create the pipe
	pipe := collection.Pipe(pipeline).AllowDiskUse()

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

//MapReduceCollection builds collections via javascript map reduce jobs
func (d *DB) MapReduceCollection(sourceCollection string, job mgo.MapReduce) bool {
	// Make a copy of the current session
	session := d.Session.Copy()
	defer session.Close()

	session.SetSocketTimeout(2 * time.Hour)

	// Identify the source collection we will aggregate information from into the new collection
	if !d.CollectionExists(sourceCollection) {
		d.resources.Log.Info("Failed map reduce: (Source collection: ", sourceCollection, " doesn't exist)")
		return false
	}
	collection := session.DB(d.selected).C(sourceCollection)

	// Map reduce that shit
	_, err := collection.Find(nil).MapReduce(&job, nil)

	// If error, Throw computer against wall and drink 2 angry beers while
	// questioning your life, purpose, and relationships.
	if err != nil {
		d.resources.Log.Error("Failed map reduce for: ", sourceCollection, err)
		return false
	}

	return true
}
