package database

import (
	// "fmt"

	"github.com/ocmdev/rita/analysis/TBD"
	"github.com/ocmdev/rita/analysis/blacklisted"
	"github.com/ocmdev/rita/analysis/scanning"
	"github.com/ocmdev/rita/analysis/urls"
	"github.com/ocmdev/rita/analysis/useragent"
	dataTBD "github.com/ocmdev/rita/datatypes/TBD"

	"gopkg.in/mgo.v2/bson"
)

///////////////////////////////////////////////////////////////////////////////
//////////////////// LAYER 2 COLLECTION BUILDING FUNCTIONS ////////////////////
///////////////////////////////////////////////////////////////////////////////
/*
 * Name:       BuildBlacklistedCollection
 * Purpose:    Builds the blacklisted collection
 * Build Type:
 * Source:
 * comments:
 */
func (d *DB) BuildBlacklistedCollection() {
	collection_name := d.r.System.BlacklistedConfig.BlacklistTable
	collection_keys := []string{"bl_hash", "host"}
	error_check := d.createCollection(collection_name, collection_keys)
	if error_check != "" {
		d.l.Error("Failed: ", collection_name, error_check)
		return
	}
	b := blacklisted.New(d.r)
	b.Run()
}

func (d *DB) BuildTBDCollection() {
	collection_name := d.r.System.TBDConfig.TBDTable
	collection_keys := []string{"uconn_id", "ts_score"}
	error_check := d.createCollection(collection_name, collection_keys)
	if error_check != "" {
		d.l.Error("Failed: ", collection_name, error_check)
		return
	}
	u := TBD.New(d.r)
	u.Run()
}

func (d *DB) GetTBDResultsView(cutoffScore float64) []dataTBD.TBDAnalysisView {
	pipeline := TBD.GetViewPipeline(d.r, cutoffScore)
	var results []dataTBD.TBDAnalysisView
	d.aggregateCollection(d.r.System.TBDConfig.TBDTable, pipeline, &results)
	return results
}

/*
 * Name:       BuildScanningCollection
 * Purpose:    Builds the scanning collection
 * Build Type: aggregation
 * Source:     connections table
 * comments:
 */
func (d *DB) BuildScanningCollection() {
	// Create the aggregate command
	source_collection_name,
		new_collection_name,
		new_collection_keys,
		pipeline := scanning.GetScanningCollectionScript(&d.r.System)

	// Create it
	error_check := d.createCollection(new_collection_name, new_collection_keys)
	if error_check != "" {
		d.l.Error("Failed: ", new_collection_name, error_check)
		return
	}

	// Aggregate it!
	results := []bson.M{}
	d.aggregateCollection(source_collection_name, pipeline, &results)
}

/*
 * Name:       BuildUrlsCollection
 * Purpose:    Builds the urls collection
 * Build Type: map reduce -> aggregation
 * Source:     http table
 * comments:
 */
func (d *DB) BuildUrlsCollection() {
	// Create the aggregate command
	source_collection_name,
		new_collection_name,
		new_collection_keys,
		job,
		pipeline := urls.GetUrlCollectionScript(&d.r.System)

	// Create it
	error_check := d.createCollection(new_collection_name, new_collection_keys)
	if error_check != "" {
		d.l.Error("Failed: ", new_collection_name, error_check)
		return
	}

	// Map reduce it!
	if !d.mapReduceCollection(source_collection_name, job) {
		return
	}

	// Aggregate it
	results := []bson.M{}
	d.aggregateCollection(new_collection_name, pipeline, &results)
}

/*
 * Name:       BuildHostnamesCollection
 * Purpose:    Builds the hostnames collection
 * Build Type: aggregation
 * Source:     urls collection
 * comments:	 Relies on the url collection being built
 */
func (d *DB) BuildHostnamesCollection() {
	source_collection_name,
		new_collection_name,
		new_collection_keys,
		pipeline := urls.GetHostnamesAggregationScript(&d.r.System)

	if !d.collectionExists(d.r.System.UrlsConfig.UrlsTable) {
		d.l.Error("The urls collection must be built before the hostnames table")
	}

	err := d.createCollection(new_collection_name, new_collection_keys)
	if err != "" {
		d.l.Error("Failed: ", new_collection_name, err)
		return
	}

	results := []bson.M{}
	d.aggregateCollection(source_collection_name, pipeline, &results)
}

/*
 * Name:       BuildUserAgentCollection
 * Purpose:    Builds the useragent collection
 * Build Type: aggregation
 * Source:     http table
 * comments:
 */
func (d *DB) BuildUserAgentCollection() {
	// Create the aggregate command
	source_collection_name,
		new_collection_name,
		new_collection_keys,
		pipeline := useragent.GetUserAgentCollectionScript(&d.r.System)

	// Create it
	error_check := d.createCollection(new_collection_name, new_collection_keys)
	if error_check != "" {
		d.l.Error("Failed: ", new_collection_name, error_check)
		return
	}

	// Aggregate it!
	results := []bson.M{}
	d.aggregateCollection(source_collection_name, pipeline, &results)
}
