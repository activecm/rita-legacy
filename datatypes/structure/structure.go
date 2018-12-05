package structure

import (
	"github.com/globalsign/mgo/bson"
)

type (
	//Host describes a computer interface found in the
	//network traffic being analyzed
	Host struct {
		ID    bson.ObjectId `bson:"_id,omitempty"`
		IP    string        `bson:"ip"`
		Local bool          `bson:"local"`
		IPv4  bool          `bson:"ipv6"`
	}

	//IPv4Binary provides a way to store a binary representation of an
	//IPv4 address in MongoDB
	IPv4Binary struct {
		ID         bson.ObjectId `bson:"_id,omitempty"`
		IP         string        `bson:"ip"`
		IPv4Binary int64         `bson:"ipv4_binary"`
	}

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

	//IPv6Binary provides a way to store a binary representation of an
	//IPv6 address in MongoDB.
	IPv6Binary struct {
		ID         bson.ObjectId `bson:"_id,omitempty"`
		IP         string        `bson:"ip"`
		IPv6Binary IPv6Integers  `bson:"ipv6_binary"`
	}

	//UniqueConnection describes a pair of computer interfaces which contacted
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
		TotalDuration   float32       `bson:"total_duration"`
		Ts              []int64       `bson:"ts_list"`         // Connection timestamps for this src, dst pair
		OrigIPBytes     []int64       `bson:"orig_bytes_list"` // Src to dst connection sizes for each connection
	}
)
