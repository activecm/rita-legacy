package blacklist

import (
	"github.com/activecm/rita-legacy/resources"
	"github.com/globalsign/mgo/bson"
)

// HostnameResults finds blacklisted hostnames in the database and the IPs of the
// hosts which connected to the blacklisted hostnames. The results will be sorted in
// descending order keyed on of {uconn_count, conn_count, total_bytes} depending on the value
// of sort. limit and noLimit control how many results are returned.
func HostnameResults(res *resources.Resources, sort string, limit int, noLimit bool) ([]HostnameResult, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	blHostsQuery := []bson.M{
		// find blacklisted hostnames and the IPs associated with them
		{"$match": bson.M{"blacklisted": true}},
		{"$project": bson.M{
			"host":    1,
			"dat.ips": 1,
		}},
		// aggregate over time/ chunks
		{"$unwind": "$dat"},
		// remove duplicate ips associated with each hostname
		{"$unwind": "$dat.ips"},
		// remove network_name as it may not be consistent with
		// network_uuid and we don't need to display it
		{"$project": bson.M{"dat.ips.network_name": 0}},
		{"$group": bson.M{
			"_id": "$host",
			"ips": bson.M{"$addToSet": "$dat.ips"},
		}},
		{"$unwind": "$ips"},
		// find out which IPs connected to each hostname via uconn
		{"$lookup": bson.M{
			"from": "uconn",
			"let":  bson.M{"ip": "$ips.ip", "network_uuid": "$ips.network_uuid"},
			"pipeline": []bson.M{{"$match": bson.M{"$expr": bson.M{
				"$and": []bson.M{
					{"$eq": []string{"$dst", "$$ip"}},
					{"$eq": []string{"$dst_network_uuid", "$$network_uuid"}},
				},
			}}}},
			"as": "uconn",
		}},
		{"$unwind": "$uconn"},
		{"$unwind": "$uconn.dat"},
		{"$project": bson.M{
			"host":             1,
			"src_ip":           "$uconn.src",
			"src_network_uuid": "$uconn.src_network_uuid",
			"src_network_name": "$uconn.src_network_name",
			"conns":            "$uconn.dat.count",
			"tbytes":           "$uconn.dat.tbytes",
		}},
		// remove duplicate source for each host and sum bytes
		// and connections per blacklisted hostname.
		// we have to do this in parts because network_name
		// may be different between IPs with the same network_uuid
		{"$group": bson.M{
			"_id": bson.M{
				"host":             "$_id",
				"src_ip":           "$src_ip",
				"src_network_uuid": "$src_network_uuid",
			},
			"src_network_name": bson.M{"$last": "$src_network_name"},
			"conns":            bson.M{"$sum": "$conns"},
			"tbytes":           bson.M{"$sum": "$tbytes"},
		}},
		{"$project": bson.M{
			"_id":    0,
			"host":   "$_id.host",
			"conns":  1,
			"tbytes": 1,
			"src": bson.M{
				"ip":           "$_id.src_ip",
				"network_uuid": "$_id.src_network_uuid",
				"network_name": "$src_network_name",
			},
		}},

		{"$group": bson.M{
			"_id":     "$host",
			"conns":   bson.M{"$sum": "$conns"},
			"tbytes":  bson.M{"$sum": "$tbytes"},
			"sources": bson.M{"$addToSet": "$src"},
		}},
		{"$project": bson.M{
			"_id":         0,
			"host":        "$_id",
			"uconn_count": bson.M{"$size": bson.M{"$ifNull": []interface{}{"$sources", []interface{}{}}}},
			"conn_count":  "$conns",
			"total_bytes": "$tbytes",
			"sources":     1,
		}},
		{"$sort": bson.M{sort: -1}},
	}

	if !noLimit {
		blHostsQuery = append(blHostsQuery, bson.M{"$limit": limit})
	}

	var blHosts []HostnameResult

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.DNS.HostnamesTable).Pipe(blHostsQuery).AllowDiskUse().All(&blHosts)

	return blHosts, err
}

// SrcIPResults finds blacklisted source IPs in the database and the IPs of the
// hosts which the blacklisted IP connected to. The results will be sorted in
// descending order keyed on of {uconn_count, conn_count, total_bytes} depending on the value
// of sort. limit and noLimit control how many results are returned.
func SrcIPResults(res *resources.Resources, sort string, limit int, noLimit bool) ([]IPResult, error) {
	return ipResults(res, sort, limit, noLimit, true)
}

// DstIPResults finds blacklisted destination IPs in the database and the IPs of the
// hosts which connected to the blacklisted IP. The results will be sorted in
// descending order keyed on of {uconn_count, conn_count, total_bytes} depending on the value
// of sort. limit and noLimit control how many results are returned.
func DstIPResults(res *resources.Resources, sort string, limit int, noLimit bool) ([]IPResult, error) {
	return ipResults(res, sort, limit, false, noLimit)
}

// ipResults implements SrcIPResults and DstIPResults. Set sourceDestFlag to true
// to find blacklisted source IPs. Set sourceDestFlag to false to find blacklisted
// destination IPs.
func ipResults(res *resources.Resources, sort string, limit int, noLimit bool, sourceDestFlag bool) ([]IPResult, error) {
	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	var hostMatch bson.M
	var blHostField string
	var blPeerField string

	if sourceDestFlag { // find blacklisted source IPs
		blHostField = "src"
		blPeerField = "dst"
		hostMatch = bson.M{
			"$and": []bson.M{
				{"blacklisted": true},
				{"dat.count_src": bson.M{"$gt": 0}},
			}}
	} else { // find blacklisted destination IPs
		blHostField = "dst"
		blPeerField = "src"
		hostMatch = bson.M{
			"$and": []bson.M{
				{"blacklisted": true},
				{"dat.count_dst": bson.M{"$gt": 0}},
			}}
	}

	var blIPs []IPResult

	blIPQuery := []bson.M{
		// find blacklisted source/ destination hosts
		{"$match": hostMatch},
		// only select ip info from hosts collection
		{"$project": bson.M{
			"ip":           1,
			"network_uuid": 1,
			"network_name": 1,
		}},
		// join on both src/dst and src/dst_network_uuid
		{"$lookup": bson.M{
			"from": "uconn",
			"let":  bson.M{"ip": "$ip", "network_uuid": "$network_uuid"},
			"pipeline": []bson.M{{"$match": bson.M{"$expr": bson.M{
				"$and": []bson.M{
					{"$eq": []string{"$" + blHostField, "$$ip"}},
					{"$eq": []string{"$" + blHostField + "_network_uuid", "$$network_uuid"}},
				},
			}}}},
			"as": "uconn",
		}},
		// convert lookup array to separate records
		{"$unwind": "$uconn"},
		// start aggregation across chunks/ time
		{"$unwind": "$uconn.dat"},
		// simplify names/ drop unused data
		{"$project": bson.M{
			"ip":                1,
			"network_uuid":      1,
			"network_name":      1,
			"peer_ip":           "$uconn." + blPeerField,
			"peer_network_uuid": "$uconn." + blPeerField + "_network_uuid",
			"peer_network_name": "$uconn." + blPeerField + "_network_name",
			"conns":             "$uconn.dat.count",
			"tbytes":            "$uconn.dat.tbytes",
		}},
		// we want to group on the blacklisted IP we started with and find
		// the set of its peer IPs. Creating a set over {ip, network_uuid, network_name} objects
		// takes a bit of work since the network_name may change over time and should not
		// be used when determining equality.
		// to get around this, we pick one of the names associated with a
		// given peer uuid and throw away the rest.
		// aggregate uconn data through time (over chunks)
		{"$group": bson.M{
			"_id": bson.M{
				// group within each blacklisted host
				"ip":           "$ip",
				"network_uuid": "$network_uuid",
				// group on the peers which connected to the blacklisted host
				"peer_ip":           "$peer_ip",
				"peer_network_uuid": "$peer_network_uuid",
			},
			// there should only be one network_name in each record
			// as it comes from the hosts collection
			"network_name": bson.M{"$last": "$network_name"},
			// use one of the network names associated with the network_uuid
			// for this partial result
			"peer_network_name": bson.M{"$last": "$peer_network_name"},
			// compute the partial sums over connections and bytes
			"conns":  bson.M{"$sum": "$conns"},
			"tbytes": bson.M{"$sum": "$tbytes"},
		}},
		// gather the peer fields so we can use addToSet
		{"$project": bson.M{
			"_id":          0, //move the id fields back out
			"ip":           "$_id.ip",
			"network_uuid": "$_id.network_uuid",
			"network_name": "$network_name",
			"peer": bson.M{
				"ip":           "$_id.peer_ip",
				"network_uuid": "$_id.peer_network_uuid",
				"network_name": "$peer_network_name",
			},
			"conns":  1,
			"tbytes": 1,
		}},
		// group the uconn data up to find which IPs peered with this blacklisted host,
		// how many connections were made, and how much data was sent in total.
		{"$group": bson.M{
			"_id": bson.M{
				"ip":           "$ip",
				"network_uuid": "$network_uuid",
				"network_name": "$network_name",
			},
			"peers":  bson.M{"$addToSet": "$peer"},
			"conns":  bson.M{"$sum": "$conns"},
			"tbytes": bson.M{"$sum": "$tbytes"},
		}},
		// move the id fields back out and add uconn_count
		{"$project": bson.M{
			"_id":          0,
			"ip":           "$_id.ip",
			"network_uuid": "$_id.network_uuid",
			"network_name": "$_id.network_name",
			"peers":        1,
			"conn_count":   "$conns",
			"uconn_count":  bson.M{"$size": bson.M{"$ifNull": []interface{}{"$peers", []interface{}{}}}},
			"total_bytes":  "$tbytes",
		}},
		{"$sort": bson.M{sort: -1}},
	}

	if !noLimit {
		blIPQuery = append(blIPQuery, bson.M{"$limit": limit})
	}

	err := ssn.DB(res.DB.GetSelectedDB()).C(res.Config.T.Structure.HostTable).Pipe(blIPQuery).AllowDiskUse().All(&blIPs)

	return blIPs, err

}
