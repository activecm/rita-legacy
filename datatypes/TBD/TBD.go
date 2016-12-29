package TBD

import (
	"gopkg.in/mgo.v2/bson"
)

type (

	/*** Layer 1 Collection Structures ***/
	TBD struct {
		ID            bson.ObjectId `bson:"_id,omitempty"`
		Src           string        `bson:"src"` // Going to remove these soon...
		Dst           string        `bson:"dst"` // Going to remove these soon...
		UconnID       bson.ObjectId `bson:"uconn_id"`
		Range         int64         `bson:"range"`
		Size          int64         `bson:"size"`
		RangeVals     string        `bson:"range_vals"`
		Fill          float64       `bson:"fill"`
		Spread        float64       `bson:"spread"`
		Sum           int64         `bson:"range_size"`
		Score         float64       `bson:"score"`
		Intervals     []int64       `bson:"intervals"`
		InvervalCount []int64       `bson:"interval_counts"`
		Tss           []int64       `bson:"tss"`
		TopInterval   int64         `bson:"most_frequent_interval"`
		TopIntervalCt int64         `bson:"most_frequent_interval_count"`
	}
)
