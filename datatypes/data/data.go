package data

type (
	// Conn provides structure for a subset of the fields in the
	// parser.Conn data structure. If fields are needed that are
	// not in this Conn structure use parser.Conn instead.
	Conn struct {
		Ts            int64   `bson:"ts,omitempty"`
		UID           string  `bson:"uid"`
		Src           string  `bson:"id_orig_h,omitempty"`
		Spt           int     `bson:"id_orig_p,omitempty"`
		Dst           string  `bson:"id_resp_h,omitempty"`
		Dpt           int     `bson:"id_resp_p,omitempty"`
		Dur           float64 `bson:"duration,omitempty"`
		Proto         string  `bson:"proto,omitempty"`
		LocalSrc      bool    `bson:"local_orig,omitempty"`
		LocalDst      bool    `bson:"local_resp,omitempty"`
		OriginIPBytes int64   `bson:"orig_ip_bytes,omitempty"`
	}

	// DNS provides structure for a subset of the fields in the
	// parser.DNS data structure. If fields are needed that are
	// not in this Conn structure use parser.DNS instead.
	DNS struct {
		Ts      int64    `bson:"ts"`
		UID     string   `bson:"uid"`
		Src     string   `bson:"id_orig_h"`
		Spt     int      `bson:"id_orig_p"`
		Dst     string   `bson:"id_resp_h"`
		Dpt     int      `bson:"id_resp_p"`
		Proto   string   `bson:"proto"`
		QType   string   `bson:"qtype_name"`
		Query   string   `bson:"query"`
		Answers []string `bson:"answers"`
	}
)
