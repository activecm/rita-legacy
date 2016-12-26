package TBD

import (
	"gopkg.in/mgo.v2/bson"
)

type (
	TBDAnalysisOutput struct {
		ID            bson.ObjectId `bson:"_id,omitempty"`
		UconnID       bson.ObjectId `bson:"uconn_id"`
		TS_skew       float64       `bson:"ts_skew"`
		TS_dispersion int64         `bson:"ts_dispersion"`
		TS_duration   float64       `bson:"ts_duration"`
		TS_dRange     int64         `bson:"ts_dRange"`
		Score         float64       `bson:"score"`
	}
)
