package parsetypes

import (
	"github.com/activecm/rita/config"
	"github.com/globalsign/mgo/bson"
)

type (
	// Uconn provides a data structure for bro's unique connection data
	Uconn struct {
		// ID is the id coming out of mongodb
		ID               bson.ObjectId `bson:"_id,omitempty"`
		Source           string        `bson:"src" bro:"id.orig_h" brotype:"addr"`
		Destination      string        `bson:"dst" bro:"id.resp_h" brotype:"addr"`
		ConnectionCount  int64         `bson:"connection_count" bro:"connection_count" brotype:"connection_count"`
		LocalSource      bool          `bson:"local_src" bro:"local_orig" brotype:"bool"`
		LocalDestination bool          `bson:"local_dst" bro:"local_resp" brotype:"bool"`
		TotalBytes       int64         `bson:"total_bytes" bro:"total_bytes" brotype:"total_bytes"`
		AverageBytes     float64       `bson:"avg_bytes" bro:"avg_bytes" brotype:"avg_bytes"`
		TSList           []int64       `bson:"ts_list" bro:"ts_list" brotype:"ts_list"`
		OrigBytesList    []int64       `bson:"orig_bytes_list" bro:"orig_bytes_list" brotype:"orig_bytes_list"`
		TotalDuration    float64       `bson:"total_duration" bro:"total_duration" brotype:"total_duration"`
		MaxDuration      float64       `bson:"max_duration" bro:"max_duration" brotype:"max_duration"`
	}
)

//TargetCollection returns the mongo collection this entry should be inserted
//into
func (in *Uconn) TargetCollection(config *config.StructureTableCfg) string {
	return config.FrequentConnTable
}

//Indices gives MongoDB indices that should be used with the collection
func (in *Uconn) Indices() []string {
	return []string{"$hashed:src", "$hashed:dst", "-connection_count"}
}
