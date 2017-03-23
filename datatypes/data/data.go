package data

import "gopkg.in/mgo.v2/bson"

type (
	// Conn provides structure for a subset of the fields in the
	// parser.Conn data structure. If fields are needed that are
	// not in this Conn structure use parser.Conn instead.
	Conn struct {
		ID       bson.ObjectId `bson:"_id,omitempty"`
		Ts       int64         `bson:"ts,omitempty"`
		UID      string        `bson:"uid"`
		Src      string        `bson:"id_orig_h,omitempty"`
		Spt      int           `bson:"id_orig_p,omitempty"`
		Dst      string        `bson:"id_resp_h,omitempty"`
		Dpt      int           `bson:"id_resp_p,omitempty"`
		Dur      float64       `bson:"duration,omitempty"`
		Proto    string        `bson:"proto,omitempty"`
		LocalSrc bool          `bson:"local_orig,omitempty"`
		LocalDst bool          `bson:"local_resp,omitempty"`
	}
)
