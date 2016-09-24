package data

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

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

	// HostIntel provides a structure for host intelligence
	// data. These structures are filled in by the HostIntel and
	// blacklisted modules.
	HostIntel struct {
		ID          bson.ObjectId `bson:"_id,omitempty"`
		Host        string        `bson:"host"`
		LastChecked time.Time     `bson:"last_check"`
		ASN         int64         `bson:"asn"`
		BGPPrefix   string        `bson:"bgp_prefix"`
		CountryCode string        `bson:"country_code"`
		Registry    string        `bson:"registry"`
		Allocated   time.Time     `bson:"allocated"`
		ASNName     string        `bson:"asn_name"`
	}

	// Conn provides structure for a subset of the fields in the
	// various module .PartObj structures. Fields were needed that are
	// not in the Conn structure, but the use of parser.Conn created
	// an import loop. Extra stuff is left in for use later.
	ConnObj struct {
		UID           string  `bson:"uid" json:"uid"`
		TS            int64   `bson:"ts" js:"ts"`
		Src           string  `bson:"id_origin_h" json:"id_origin_h"`
		Dst           string  `bson:"id_resp_h" json:"id_resp_h"`
		SrcPort       int     `bson:"id_origin_p" json:"id_origin_p"`
		DstPort       int     `bson:"id_resp_p" json:"id_resp_p"`
		Duration      float64 `bson:"duration" json:"duration"`
		LocalSrc      bool    `bson:"local_orig" json:"local_orig"`
		LocalDst      bool    `bson:"local_resp" json:"local_resp"`
		SrcBytes      int64   `bson:"orig_bytes" json:"orig_bytes"`
		DstBytes      int64   `bson:"resp_bytes" json:"resp_bytes"`
		Proto         string  `bson:"proto" json:"proto"`
		Service       string  `bson:"service" json:"service"`
		ConnState     string  `bson:"conn_state" json:"conn_state"`
		MissedBytes   int64   `bson:"missed_bytes" json:"missed_bytes"`
		History       string  `bson:"history" json:"history"`
		OrigPkts      int64   `bson:"orig_pkts" json:"orig_pkts"`
		OrigIPBytes   int64   `bson:"orig_ip_bytes" json:"orig_ip_bytes"`
		RespPkts      int64   `bson:"resp_pkts" json:"resp_pkts"`
		RespIPBytes   int64   `bson:"resp_ip_bytes" json:"resp_ip_bytes"`
		TunnelParents string  `bson:"tunnel_parents" json:"tunnel_parents"`
	}
)
