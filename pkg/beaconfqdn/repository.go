package beaconfqdn

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/hostname"
	"github.com/globalsign/mgo/bson"
)

type (

	// Repository for host collection
	Repository interface {
		CreateIndexes() error
		Upsert(hostnameMap map[string]*hostname.Input)
	}

	updateInfo struct {
		selector bson.M
		query    bson.M
	}

	//update ....
	update struct {
		beacon     updateInfo
		hostIcert  updateInfo
		hostBeacon updateInfo
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

	//Result represents a beacon between two hosts. Contains information
	//on connection delta times and the amount of data transferred
	Result struct {
		data.UniqueIPPair `bson:",inline"`
		Connections       int64   `bson:"connection_count"`
		AvgBytes          float64 `bson:"avg_bytes"`
		Ts                TSData  `bson:"ts"`
		Ds                DSData  `bson:"ds"`
		Score             float64 `bson:"score"`
	}

	//StrobeResult represents a unique connection with a large amount
	//of connections between the hosts
	StrobeResult struct {
		data.UniqueIPPair `bson:",inline"`
		ConnectionCount   int64 `bson:"connection_count"`
	}

	//uniqueSrcHostnamePair ...
	uniqueSrcHostnamePair struct {
		SrcIP          string      `bson:"src"`
		SrcNetworkUUID bson.Binary `bson:"src_network_uuid"`
		FQDN           string      `bson:"fqdn"`
	}
)

//newUniqueSrcHostnamePair binds a pair of UniqueIPs where direction matters.
func newUniqueSrcHostnamePair(source data.UniqueIP, fqdn string) uniqueSrcHostnamePair {
	return uniqueSrcHostnamePair{
		SrcIP:          source.IP,
		SrcNetworkUUID: source.NetworkUUID,
		FQDN:           fqdn,
	}
}

//BSONKey generates a BSON map which may be used to index a given a unique src
// fqdn pair
//Includes IP and Network UUID.
func (p uniqueSrcHostnamePair) BSONKey() bson.M {
	key := bson.M{
		"src":              p.SrcIP,
		"src_network_uuid": p.SrcNetworkUUID,
		"fqdn":             p.FQDN,
	}
	return key
}
