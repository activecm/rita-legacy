package structure

import (
	"github.com/globalsign/mgo/bson"
)

type (

	//IPv6Integers provides a way to store a binary representation of an
	//IPv6 address in MongoDB. The 128 bit address is split into four 32 bit
	//values. However, MongoDB cannot store unsigned numbers, so we use 64 bit
	//integers to hold the values.
	IPv6Integers struct {
		I1 int64 `bson:"1"`
		I2 int64 `bson:"2"`
		I3 int64 `bson:"3"`
		I4 int64 `bson:"4"`
	}

	//Host describes an IP address found in the
	//network traffic being analyzed
	Host struct {
		ID         bson.ObjectId `bson:"_id,omitempty"`
		IP         string        `bson:"ip"`
		Local      bool          `bson:"local"`
		IPv4       bool          `bson:"ipv4"`
		CountSrc   int32         `bson:"count_src"`
		CountDst   int32         `bson:"count_dst"`
		IPv4Binary int64         `bson:"ipv4_binary"`
		// IPv6Binary IPv6Integers  `bson:"ipv6_binary"` // for future ipv6 support
		MaxDuration float32 `bson:"max_duration"`
	}

	//UniqueConnection describes a pair of IP addresses which contacted
	//each other over the observation period
	UniqueConnection struct {
		ID              bson.ObjectId `bson:"_id,omitempty"`
		ConnectionCount int           `bson:"connection_count"`
		Src             string        `bson:"src"`
		Dst             string        `bson:"dst"`
		LocalSrc        bool          `bson:"local_src"`
		LocalDst        bool          `bson:"local_dst"`
		TotalBytes      int           `bson:"total_bytes"`
		AverageBytes    float32       `bson:"average_bytes"`
		TsList          []int64       `bson:"ts_list"`         // Connection timestamps for this src, dst pair
		OrigIPBytes     []int64       `bson:"orig_bytes_list"` // Src to dst connection sizes for each connection
		MaxDuration     float32       `bson:"max_duration"`
	}
)
