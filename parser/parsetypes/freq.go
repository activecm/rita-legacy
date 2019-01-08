package parsetypes

import (
	"github.com/activecm/rita/config"
)

type (
	// Freq provides a data structure for bro's connection data
	Freq struct {
		// ID is the id coming out of mongodb
		// ID              bson.ObjectId `bson:"_id,omitempty"`
		Source          string `bson:"src" bro:"id.orig_h" brotype:"addr"`
		Destination     string `bson:"dst" bro:"id.resp_h" brotype:"addr"`
		ConnectionCount int64  `bson:"connection_count" bro:"connection_count" brotype:"connection_count"`
	}
)

//TargetCollection returns the mongo collection this entry should be inserted
//into
func (in *Freq) TargetCollection(config *config.StructureTableCfg) string {
	return config.FrequentConnTable
}

//Indices gives MongoDB indices that should be used with the collection
func (in *Freq) Indices() []string {
	return []string{"$hashed:src", "$hashed:dst", "-connection_count"}
}
