package sniconn

import "github.com/activecm/rita/pkg/data"

type TLSInput struct {
	Hosts data.UniqueSrcFQDNPair

	IsLocalSrc bool

	ConnectionCount int64
	Timestamps      data.Int64Set
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
	Timestamps      data.Int64Set
	RespondingIPs   data.UniqueIPSet
	RespondingPorts data.IntSet

	ZeekUIDs []string
}

// there is no way in the conn parser to know if the conn parser needs
// to link the data size in with the sni data

// if we guarantee that the http and ssl logs are imported first, then we can
// track the uids that make up

/*

ssl log links a timestamp, uuid, conn tuple, sni
{
	src:
	src_network_uuid:
	src_network_name:
	fqdn:

	beacon: {
		ts: {

		}

		ds: {

		}
		score:

		connection_count:
		avg_bytes:
	}

	dat: []{
		http_host: {
			ts: []
			bytes: []
			count:
			tbytes:
			cid:
			dst_ips: []
			dst_ports: []

			// uris would be nice to have but too big. top 100 uris would be best, but not in scope
			// user_agents: []
			// methods: []
		}
		tls_sni: {
			ts: []
			bytes: []
			count:
			tbytes:
			cid:
			dst_ips: []
			dst_ports: []

			// subjects: []
			// ja3s: []
		}



	}
}
*/
