package database

///////////////////////////////////////////////////////////////////////////////
//////////////////// LAYER 1 COLLECTION BUILDING FUNCTIONS ////////////////////
///////////////////////////////////////////////////////////////////////////////

// BuildConnectionsCollection builds the 'conn' collection. Sourced from the
// bro parser.
func buildConnectionsCollection(res *Resources) {
	collection_name := res.System.StructureConfig.ConnTable
	collection_keys := []string{"$hashed:id_origin_h", "$hashed:id_resp_h", "$hashed:uid", "-duration"}
	error_check := res.DB.CreateCollection(collection_name, collection_keys)
	if error_check != "" {
		res.Log.Error("Failed: ", collection_name, error_check)
		return
	}
}

// BuildHttpCollection builds the 'http' collection. Sourced from the bro parser.
func buildHttpCollection(res *Resources) {
	collection_name := res.System.StructureConfig.HTTPTable
	collection_keys := []string{"$hashed:uid"}
	error_check := res.DB.CreateCollection(collection_name, collection_keys)
	if error_check != "" {
		res.Log.Error("Failed: ", collection_name, error_check)
		return
	}
}

// BuildHttpCollection builds the 'http' collection. Sourced from the bro parser.
func buildDNSCollection(res *Resources) {
	collection_name := res.System.StructureConfig.DNSTable
	collection_keys := []string{"$hashed:uid"}
	error_check := res.DB.CreateCollection(collection_name, collection_keys)
	if error_check != "" {
		res.Log.Error("Failed: ", collection_name, error_check)
		return
	}
}
