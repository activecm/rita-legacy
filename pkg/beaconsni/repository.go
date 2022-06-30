package beaconsni

import (
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/sniconn"
	"github.com/globalsign/mgo"
)

// Repository for beaconsni collection
type Repository interface {
	CreateIndexes() error
	Upsert(tlsMap map[string]*sniconn.TLSInput, httpMap map[string]*sniconn.HTTPInput, hostMap map[string]*host.Input, minTimestamp, maxTimestamp int64)
}

type mgoBulkAction func(*mgo.Bulk) int

type mgoBulkActions map[string]mgoBulkAction

type dissectorResults struct {
	Hosts           data.UniqueSrcFQDNPair
	ConnectionCount int64
	TotalBytes      int64
	TsList          []int64
	OrigBytesList   []int64
}
