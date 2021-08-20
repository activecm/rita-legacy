package beaconproxy

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/uconnproxy"
	"github.com/globalsign/mgo/bson"
)

type (

	// Repository for host collection
	Repository interface {
		CreateIndexes() error
		Upsert(uconnProxyMap map[string]*uconnproxy.Input, minTimestamp, maxTimestamp int64)
	}

	updateInfo struct {
		selector bson.M
		query    bson.M
	}

	//update ....
	update struct {
		beacon     updateInfo
		hostBeacon updateInfo
		uconnproxy updateInfo
	}

	//TSData ...
	TSData struct {
		Range      int64   `bson:"range"`
		Mode       int64   `bson:"mode"`
		ModeCount  int64   `bson:"mode_count"`
		Skew       float64 `bson:"skew"`
		Dispersion int64   `bson:"dispersion"`
	}

	//Result represents a beacon proxy between a source IP and
	// an fqdn.
	Result struct {
		FQDN           string        `bson:"fqdn"`
		SrcIP          string        `bson:"src"`
		SrcNetworkName string        `bson:"src_network_name"`
		SrcNetworkUUID bson.Binary   `bson:"src_network_uuid"`
		Connections    int64         `bson:"connection_count"`
		Ts             TSData        `bson:"ts"`
		Score          float64       `bson:"score"`
		Proxy          data.UniqueIP `bson:"proxy"`
	}

	//StrobeResult represents a unique connection with a large amount
	//of connections between the hosts
	StrobeResult struct {
		data.UniqueSrcFQDNPair `bson:",inline"`
		ConnectionCount        int64 `bson:"connection_count"`
	}
)
