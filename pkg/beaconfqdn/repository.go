package beaconfqdn

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/host"
	"github.com/globalsign/mgo/bson"
)

type (

	// Repository for host collection
	Repository interface {
		CreateIndexes() error
		Upsert(hostMap map[string]*host.Input, minTimestamp, maxTimestamp int64)
	}

	update struct {
		selector bson.M
		query    bson.M
	}

	// hostnameIPs is used with reverseDNSQueryWithIPs() in order to read in records
	// from the `hostnames` collection via a MongoDB aggregation
	hostnameIPs struct {
		Host        string          `bson:"_id"`
		ResolvedIPs []data.UniqueIP `bson:"ips"`
	}

	//fqdnInput represents intermediate state required to perform fqdn beaconing analysis
	fqdnInput struct {
		FQDN            string           //A hostname
		Src             data.UniqueSrcIP // Single src that connected to a hostname
		ResolvedIPs     []data.UniqueIP  //Set of resolved UniqueIPs associated with a given hostname
		InvalidCertFlag bool
		ConnectionCount int64
		TotalBytes      int64
		TsList          []int64
		OrigBytesList   []int64
		DstBSONList     []bson.M // set of resolved UniqueDstIPs since we need it in that format
	}

	//TSData ...
	TSData struct {
		Range      int64   `bson:"range"`
		Mode       int64   `bson:"mode"`
		ModeCount  int64   `bson:"mode_count"`
		Skew       float64 `bson:"skew"`
		Dispersion int64   `bson:"dispersion"`
		Duration   float64 `bson:"duration"`
	}

	//DSData ...
	DSData struct {
		Skew       float64 `bson:"skew"`
		Dispersion int64   `bson:"dispersion"`
		Range      int64   `bson:"range"`
		Mode       int64   `bson:"mode"`
		ModeCount  int64   `bson:"mode_count"`
	}

	//Result represents a beacon FQDN between a source IP and
	// an FQDN. An FQDN can be comprised of one or more destination IPs.
	// Contains information on connection delta times and the amount of data transferred
	Result struct {
		FQDN           string          `bson:"fqdn"`
		SrcIP          string          `bson:"src"`
		SrcNetworkName string          `bson:"src_network_name"`
		SrcNetworkUUID bson.Binary     `bson:"src_network_uuid"`
		Connections    int64           `bson:"connection_count"`
		AvgBytes       float64         `bson:"avg_bytes"`
		Ts             TSData          `bson:"ts"`
		Ds             DSData          `bson:"ds"`
		Score          float64         `bson:"score"`
		ResolvedIPs    []data.UniqueIP `bson:"resolved_ips"`
	}

	//StrobeResult represents a unique connection with a large amount
	//of connections between the hosts
	StrobeResult struct {
		data.UniqueSrcFQDNPair `bson:",inline"`
		ConnectionCount        int64 `bson:"connection_count"`
	}
)
