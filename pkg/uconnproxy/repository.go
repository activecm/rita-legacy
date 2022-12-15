package uconnproxy

import (
	"github.com/activecm/rita/pkg/data"
)

// Repository for uconnproxy collection
type Repository interface {
	CreateIndexes() error
	Upsert(uconnProxyMap map[string]*Input)
}

// Input structure for sending data
// to the analyzer. Contains a tuple of
// Src IP/UUID/Name and an FQDN to which the Src IP
// was attempting to communicate.
// Contains a list of unique time stamps for the
// connections out from the Src to the FQDN via the
// proxy server and a count of the connections.
type Input struct {
	Hosts           data.UniqueSrcFQDNPair
	TsList          []int64
	TsListFull      []int64
	Proxy           data.UniqueIP
	ConnectionCount int64
}
