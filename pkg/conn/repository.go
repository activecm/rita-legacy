package conn

import (
	"github.com/activecm/rita/parser/parsetypes"
)

// Repository for conn collection
type Repository interface {
	BulkDelete(conns []*parsetypes.Conn) error
}

// AnalysisView provides structure for a subset of the fields in the
// parser.Conn data structure. If fields are needed that are
// not in this Conn structure use parser.Conn instead.
type AnalysisView struct {
	Ts              int64   `bson:"ts,omitempty"`
	UID             string  `bson:"uid"`
	Src             string  `bson:"id_orig_h,omitempty"`
	Spt             int     `bson:"id_orig_p,omitempty"`
	Dst             string  `bson:"id_resp_h,omitempty"`
	Dpt             int     `bson:"id_resp_p,omitempty"`
	Dur             float64 `bson:"duration,omitempty"`
	Proto           string  `bson:"proto,omitempty"`
	LocalSrc        bool    `bson:"local_orig,omitempty"`
	LocalDst        bool    `bson:"local_resp,omitempty"`
	OriginIPBytes   int64   `bson:"orig_ip_bytes,omitempty"`
	OriginPackets   int64   `bson:"orig_pkts,omitempty"`
	ResponsePackets int64   `bson:"resp_pkts,omitempty"`
}
