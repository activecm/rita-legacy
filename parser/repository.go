package parser

import (
	"sync"

	"github.com/activecm/rita/pkg/certificate"
	"github.com/activecm/rita/pkg/data"
	"github.com/activecm/rita/pkg/host"
	"github.com/activecm/rita/pkg/hostname"
	"github.com/activecm/rita/pkg/sniconn"
	"github.com/activecm/rita/pkg/uconn"
	"github.com/activecm/rita/pkg/uconnproxy"
	"github.com/activecm/rita/pkg/useragent"
)

// ParseResults contains the data which the analysis packages
// expect from the parser as well as locks for safely
// accessing the data from multipel goroutines.
type ParseResults struct {
	UniqueConnMap       map[string]*uconn.Input
	UniqueConnLock      *sync.Mutex
	ProxyUniqueConnMap  map[string]*uconnproxy.Input
	ProxyUniqueConnLock *sync.Mutex
	HostMap             map[string]*host.Input
	HostLock            *sync.Mutex
	HostnameMap         map[string]*hostname.Input
	HostnameLock        *sync.Mutex
	UseragentMap        map[string]*useragent.Input
	UseragentLock       *sync.Mutex
	CertificateMap      map[string]*certificate.Input
	CertificateLock     *sync.Mutex
	ExplodedDNSMap      map[string]int
	ExplodedDNSLock     *sync.Mutex
	TLSConnMap          map[string]*sniconn.TLSInput
	TLSConnLock         *sync.Mutex
	HTTPConnMap         map[string]*sniconn.HTTPInput
	HTTPConnLock        *sync.Mutex
	ZeekUIDMap          map[string]*data.ZeekUIDRecord
	ZeekUIDLock         *sync.Mutex
}

// newParseResults instantiates a ParseResults struct
func newParseResults() ParseResults {
	return ParseResults{
		UniqueConnMap:       make(map[string]*uconn.Input),
		UniqueConnLock:      new(sync.Mutex),
		ProxyUniqueConnMap:  make(map[string]*uconnproxy.Input),
		ProxyUniqueConnLock: new(sync.Mutex),
		HostMap:             make(map[string]*host.Input),
		HostLock:            new(sync.Mutex),
		HostnameMap:         make(map[string]*hostname.Input),
		HostnameLock:        new(sync.Mutex),
		UseragentMap:        make(map[string]*useragent.Input),
		UseragentLock:       new(sync.Mutex),
		CertificateMap:      make(map[string]*certificate.Input),
		CertificateLock:     new(sync.Mutex),
		ExplodedDNSMap:      make(map[string]int),
		ExplodedDNSLock:     new(sync.Mutex),
		TLSConnMap:          make(map[string]*sniconn.TLSInput),
		TLSConnLock:         new(sync.Mutex),
		HTTPConnMap:         make(map[string]*sniconn.HTTPInput),
		HTTPConnLock:        new(sync.Mutex),
		ZeekUIDMap:          make(map[string]*data.ZeekUIDRecord),
		ZeekUIDLock:         new(sync.Mutex),
	}
}
