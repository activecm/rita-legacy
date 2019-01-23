package parsetypes

import (
	"github.com/globalsign/mgo/bson"
)

type (
	// Host collection contains the unique ip addresses found in conns
	Host struct {
		ID         bson.ObjectId `bson:"_id,omitempty"`
		IP         string        `bson:"ip"`
		Local      bool          `bson:"local"`
		IPv4       bool          `bson:"ipv4"`
		CountSrc   int32         `bson:"count_src"`
		CountDst   int32         `bson:"count_dst"`
		IPv4Binary int64         `bson:"ipv4_binary"`
		// IPv6Binary IPv6Integers  `bson:"ipv6_binary"` // for future ipv6 support
		MaxDuration        float32 `bson:"max_duration"`
		MaxBeaconScore     float64 `bson:"max_beacon_score"`
		MaxBeaconConnCount int     `bson:"max_beacon_conn_count"`
		BlOutCount         int32   `bson:"bl_out_count"`
		BlInCount          int32   `bson:"bl_in_count"`
		BlSumAvgBytes      int32   `bson:"bl_sum_avg_bytes"`
		BlTotalBytes       int32   `bson:"bl_total_bytes"`
		TxtQueryCount      int     `bson:"txt_query_count"`
	}
)
