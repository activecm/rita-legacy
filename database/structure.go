package database

import (
	// "fmt"

	"github.com/ocmdev/rita/analysis/structure"

	// log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
)

///////////////////////////////////////////////////////////////////////////////
//////////////////// LAYER 1 COLLECTION BUILDING FUNCTIONS ////////////////////
///////////////////////////////////////////////////////////////////////////////

// BuildConnectionsCollection builds the 'conn' collection. Sourced from the
// bro parser.
func (d *DB) BuildConnectionsCollection() {
	collection_name := d.r.System.StructureConfig.ConnTable
	collection_keys := []string{"$hashed:id_origin_h", "$hashed:id_resp_h", "$hashed:uid", "-duration"}
	error_check := d.createCollection(collection_name, collection_keys)
	if error_check != "" {
		d.l.Error("Failed: ", collection_name, error_check)
		return
	}
}

// BuildHttpCollection builds the 'http' collection. Sourced from the bro parser.
func (d *DB) BuildHttpCollection() {
	collection_name := d.r.System.StructureConfig.HttpTable
	collection_keys := []string{"$hashed:uid"}
	error_check := d.createCollection(collection_name, collection_keys)
	if error_check != "" {
		d.l.Error("Failed: ", collection_name, error_check)
		return
	}
}

// BuildUniqeConnectionsCollection builds the 'uconn' collection. Runs via
// mongodb aggreggation. Sourced from the 'conn' collection.
func (d *DB) BuildUniqueConnectionsCollection() {
	// Create the aggregate command
	source_collection_name,
		new_collection_name,
		new_collection_keys,
		pipeline := structure.GetUniqueConnectionsScript(&d.r.System)

	// Aggregate it!
	error_check := d.createCollection(new_collection_name, new_collection_keys)
	if error_check != "" {
		d.l.Error("Failed: ", new_collection_name, error_check)
		return
	}

	// In case we need results
	results := []bson.M{}
	d.aggregateCollection(source_collection_name, pipeline, &results)
}

// BuildHostsCollection builds the 'host' collection for this timeframe. Note
// that this is a different host collection that the one found in HostsIntelDB.
// This host collection references only hosts found in this time frame, info
// from the HostsIntelDB collection can be found by following the 'intelid' field
// after it is populated by the cymru and blacklist modules. Runs via mongodb
// aggregation. Sourced from the 'conn' table.
// TODO: Confirm that this section of code is not faster than an aggregation from
// the 'uconn' table which should have less repeated data.
func (d *DB) BuildHostsCollection() {
	// Create the aggregate command
	source_collection_name,
		new_collection_name,
		new_collection_keys,
		pipeline := structure.GetHosts(&d.r.System)

	// Aggregate it!
	error_check := d.createCollection(new_collection_name, new_collection_keys)
	if error_check != "" {
		d.l.Error("Failed: ", new_collection_name, error_check)
		return
	}

	// In case we need results
	results := []bson.M{}
	d.aggregateCollection(source_collection_name, pipeline, &results)
}
