package parsetypes

import (
	"github.com/activecm/rita/config"
	"github.com/globalsign/mgo/bson"
)

type (
	// Temp provides a data structure for bro's connection data
	Temp struct {
		// ID is the id coming out of mongodb
		ID              bson.ObjectId `bson:"_id,omitempty"`
		Source          string        `bson:"id_orig_h" bro:"id.orig_h" brotype:"addr"`
		Destination     string        `bson:"id_resp_h" bro:"id.resp_h" brotype:"addr"`
		ConnectionCount int32         `bson:"connection_count" bro:"connection_count" brotype:"connection_count"`
	}
)

//TargetCollection returns the mongo collection this entry should be inserted
//into
func (in *Temp) TargetCollection(config *config.StructureTableCfg) string {
	return config.TempTable
}

//Indices gives MongoDB indices that should be used with the collection
func (in *Temp) Indices() []string {
	return []string{"$hashed:src", "$hashed:dst", "-connection_count"}
}
