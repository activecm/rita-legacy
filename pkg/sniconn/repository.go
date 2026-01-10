package sniconn

import (
	"github.com/activecm/rita-legacy/pkg/data"
	"github.com/activecm/rita-legacy/pkg/host"
)

// Repository for uconn collection
type Repository interface {
	CreateIndexes() error
	Upsert(tlsMap map[string]*TLSInput, httpMap map[string]*HTTPInput, zeekUIDMap map[string]*data.ZeekUIDRecord, hostMap map[string]*host.Input)
}

type linkedInput struct {
	TLS            *TLSInput
	TLSZeekRecords []*data.ZeekUIDRecord

	HTTP            *HTTPInput
	HTTPZeekRecords []*data.ZeekUIDRecord
}

type TLSInput struct {
	Hosts data.UniqueSrcFQDNPair

	IsLocalSrc bool

	ConnectionCount int64
	Timestamps      []int64
	RespondingIPs   data.UniqueIPSet
	RespondingPorts data.IntSet

	RespondingCertInvalid bool
	Subjects              data.StringSet
	JA3s                  data.StringSet
	JA3Ss                 data.StringSet

	ZeekUIDs []string
}

type HTTPInput struct {
	Hosts data.UniqueSrcFQDNPair

	IsLocalSrc bool

	ConnectionCount int64
	Timestamps      []int64
	RespondingIPs   data.UniqueIPSet
	RespondingPorts data.IntSet

	Methods    data.StringSet
	UserAgents data.StringSet

	ZeekUIDs []string
}
