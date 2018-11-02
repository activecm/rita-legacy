package beacon

import (
	"github.com/globalsign/mgo/bson"
)

type (
	//AnalysisOutput contains the summary statistics of a unique connection
	AnalysisOutput struct {
		UconnID          bson.ObjectId `bson:"uconn_id"`
		TSIRange         int64         `bson:"ts_iRange"`
		TSIMode          int64         `bson:"ts_iMode"`
		TSIModeCount     int64         `bson:"ts_iMode_count"`
		TSIntervals      []int64       `bson:"ts_intervals"`
		TSIntervalCounts []int64       `bson:"ts_interval_counts"`
		TSIDispersion    int64         `bson:"ts_iDispersion"`
		TSISkew          float64       `bson:"ts_iSkew"`
		TSDuration       float64       `bson:"ts_duration"`
		TSScore          float64       `bson:"ts_score"`
		DSRange          int64         `bson:"ds_range"`
		DSMode           int64         `bson:"ds_mode"`
		DSModeCount      int64         `bson:"ds_mode_count"`
		DSSizes          []int64       `bson:"ds_sizes"`
		DSSizeCounts     []int64       `bson:"ds_counts"`
		DSDispersion     int64         `bson:"ds_dispersion"`
		DSSkew           float64       `bson:"ds_skew"`
		DSScore          float64       `bson:"ds_score"`
		Score            float64       `bson:"score"`
	}

	//AnalysisView used in order to join the uconn and beacon tables
	AnalysisView struct {
		Src           string  `bson:"src"`
		Dst           string  `bson:"dst"`
		LocalSrc      bool    `bson:"local_src"`
		LocalDst      bool    `bson:"local_dst"`
		Connections   int64   `bson:"connection_count"`
		AvgBytes      float64 `bson:"avg_bytes"`
		TSIRange      int64   `bson:"ts_iRange"`
		TSIMode       int64   `bson:"ts_iMode"`
		TSIModeCount  int64   `bson:"ts_iMode_count"`
		TSISkew       float64 `bson:"ts_iSkew"`
		TSIDispersion int64   `bson:"ts_iDispersion"`
		TSDuration    float64 `bson:"ts_duration"`
		Score         float64 `bson:"score"`
		DSSkew        float64 `bson:"ds_skew"`
		DSDispersion  int64   `bson:"ds_dispersion"`
		DSRange       int64   `bson:"ds_range"`
		DSMode        int64   `bson:"ds_mode"`
		DSModeCount   int64   `bson:"ds_mode_count"`
	}
)
