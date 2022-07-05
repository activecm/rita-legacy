## SNI Connections Package

*Documented on July 1, 2022*

---
This package records the details of the connections made between IP addresses and server name indicators (SNIs). Server name indicators are pulled from application layer (TCP/IP) protocols and usually fully qualified domain names (FQDNs). RITA refers to the set of connections from one host to a collection of others related by a shared SNI using the term "SNI connection" or "sniconn" for short.

This package records the following:
- Each source IP address and destination SNI that communicated
- How many times the source IP connected to the SNI using HTTP and TLS
    - SNI connections with connectino counts exceeding the limit defined in the RTIA configuration are marked as "strobes"
- The total amount of time the source was connected to the SNI using HTTP and TLS
- The total amount of bytes sent between the pair over HTTP and TLS
- Timestamps of the individual HTTP and TLS connections
- The amount of data sent from the source IP address to the SNI in each of the individual HTTP and TLS connections
- The destination IP addresses and ports of the HTTP and TLS servers responding to the SNI
- The certificate subjects presented by the TLS servers responding to the SNI
- The JA3 hashes of the TLS configurations sent from the source IP address to the TLS servers responding to the SNI
- The JA3S hashes of the TLS configurations presented by the TLS servers responding to the SNI
- The HTTP methods of the requests sent from the source IP address to the HTTP servers responding to the SNI
- The HTTP user agents of the requests sent from the source IP address to the HTTP servers responding to the SNI


## Package Outputs

## Source Unique IP, Destination SNI Pair
Inputs:
- `ParseResults.TLSConnMap` created by `FSImporter`
    - Field: `Hosts`
        - Type: data.UniqueSrcFQDNPair
- `ParseResults.HTTPConnMap` created by `FSImporter`
    - Field: `Hosts`
        - Type: data.UniqueSrcFQDNPair

Outputs:
- MongoDB `SNIconn` collection:
    - Field: `src`
        - Type: string
    - Field: `src_network_uuid`
        - Type: UUID
    - Field: `src_network_name`
        - Type: string
    - Field: `fqdn`
        - Type: string

The `src` field records the string representation of the IP address of the source of a SNI connection as seen in the network logs. Similarly, the field `fqdn` specifies the server name indicator/ fully qualified domain name of the destination of a SNI connection as seen in the network logs. The `src_network_uuid` and `src_network_name` fields have been introduced to disambiguate hosts using the same private IP address on separate networks.

These fields are used to select an individual entry in the `SNIconn` collection. All of the other outputs described here use the `src`, `src_network_uuid`, and `fqdn` fields as selectors when updating `SNIconn` collection entries in MongoDB.

### Chunk ID
Inputs:
- `Config.S.Rolling.CurrentChunk`
    - Type: int

Outputs:
- MongoDB `SNIconn` collection:
    - Field: `cid`
        - Type: int

The `cid` field records the chunk ID of the import session in which this unique connection document was last updated. This field is used to support rolling imports.

### TLS Destination IP Addresses and Ports
Inputs:
- `ParseResults.TLSConnMap` created by `FSImporter`
    - Field: `RespondingIPs`
        - Type: data.UniqueIPSet
    - Field: `RespondingPorts`
        - Type: data.IntSet

Outputs:
- MongoDB `SNIconn` collection:
    - Array Field: `dat`
        - Object Field: `tls`
            - Array Field: `dst_ips`
                - Field: `ip`
                    - Type: string
                - Field: `network_uuid`
                    - Type: UUID
                - Field: `network_name`
                    - Type: string
            - Array Field: `dst_ports`
                - Type: int
            - Field: `cid`
                - Type int

A new `tls` subdocument is stored in the `dat` array during each import session.

The IP addresses of the TLS servers seen responding to the SNI are stored in the `tls.dst_ips` array.

The ports on which the TLS servers responded to the SNI are stored in the `tls.dst_ports` array.

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations.

### TLS Strobe Designation
Inputs:
- `Config.S.Strobe.ConnectionLimit`
    - Type: int
- `ParseResults.TLSConnMap` created by `FSImporter`
    - Field: `ConnectionCount`
        - Type: int

Outputs:
- MongoDB `SNIconn` collection:
    - Array Field: `dat`
        - Object Field: `tls`
            - Field: `strobe`
                - Type: bool

This field is included in same `dat.tls` subdocument as the destination IP addresses described above.

If the number of TLS connections from the source to the destination in the set of network logs under consideration is greater than the strobe connection limit, the SNI connection is marked as a strobe. These hosts can be considered to have been in constant communication.

### TLS Connection Statistics
Inputs:
- `ParseResults.TLSConnMap` created by `FSImporter`
    - Field: `ConnectionCount`
        - Type: int
    - Field: `ZeekUIDs`
        - Type: []string
- `ParseResults.ZeekUIDMap` created by `FSImporter`
    - Struct Field: `Conn`
        - Field: `OrigBytes`
            - Type: int64
        - Field: `RespBytes`
            - Type: int64
        - Field: `Duration`
            - Type: float64

Outputs:
- MongoDB `SNIconn` collection:
    - Array Field: `dat`
        - Object Field: `tls`
            - Field: `count`
                - Type: int
            - Field: `tbytes`
                - Type: int
            - Field: `tdur`
                - Type: float64

These fields are included in same `dat.tls` subdocument as the destination IP addresses described above.

The number of connections from the from the source IP address to the destination TLS SNI is stored in the `count` field.

The total number of bytes sent from the source to the destination is summed together with the number of bytes sent back to the source from the destination and stored in the `tbytes` field.

The total duration of the connection from the source to the destination is stored in the `tdur` field. These duration fields are used to support long connection analysis.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the total connection count, total bytes, all of the `dat.tls` subdocuments must be summed together.

### TLS Connection Timestamps and Originating Bytes
Inputs:
- `ParseResults.TLSConnMap` created by `FSImporter`
    - Field: `Timestamps`
        - Type: data.IntSet
    - Field: `ZeekUIDs`
        - Type: []string
- `ParseResults.ZeekUIDMap` created by `FSImporter`
    - Struct Field: `Conn`
        - Field: `OrigBytes`
            - Type: int64

Outputs:
- MongoDB `SNIconn` collection:
    - Array Field: `dat`
        - Object Field: `tls`
            - Array Field: `bytes`
                - Type: int
            - Array Field: `ts`
                - Type: int

These fields are included in same `dat.tls` subdocument as the destination IP addresses described above.

The individual timestamps of the connections from the source to the destination are unioned together and stored in MongoDB. Additionally, the number of bytes the source sent to the destination in each of the connections is stored. The `beaconSNI` package takes these outputs as input.

In order to gather all of the connection timestamps across chunked imports, the `ts` arrays from each of the `dat.tls` documents must be unioned together. Similarly, in order to gather all of the data sizes across chunked imports, the `bytes` arrays from each of the `dat.tls` subdocuments must be concatenated.

If a connection is marked as a strobe, these fields may be missing or empty.

### TLS Connection Details
Inputs:
- `ParseResults.TLSConnMap` created by `FSImporter`
    - Field: `Subjects`
        - Type: data.StringSet
    - Field: `JA3s`
        - Type: data.StringSet
    - Field: `JA3Ss`
        - Type: data.StringSet

Outputs:
- MongoDB `SNIconn` collection:
    - Array Field: `dat`
        - Object Field: `tls`
            - Array Field: `subjects`
                - Type: string
            - Array Field: `ja3`
                - Type: string
            - Array Field: `ja3s`
                - Type: string

These fields are included in same `dat.tls` subdocument as the destination IP addresses described above.

The subject lines from the TLS certificates presented by the TLS servers respnding to the SNI are stored in the `subjects` field.

The JA3 hashes derived from the TLS stacks used by the client are stored in the `ja3` field.

Similarly, the JA3S hashes derived from the TLS stacks used by the TLS servers are stored in the `ja3s` field.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the whole set of TLS subjects, JA3 hashes, or JA3S hashes, these arrays in the `tls` subdocuments must be unioned together.

### HTTP Destination IP Addresses and Ports
Inputs:
- `ParseResults.HTTPConnMap` created by `FSImporter`
    - Field: `RespondingIPs`
        - Type: data.UniqueIPSet
    - Field: `RespondingPorts`
        - Type: data.IntSet

Outputs:
- MongoDB `SNIconn` collection:
    - Array Field: `dat`
        - Object Field: `http`
            - Array Field: `dst_ips`
                - Field: `ip`
                    - Type: string
                - Field: `network_uuid`
                    - Type: UUID
                - Field: `network_name`
                    - Type: string
            - Array Field: `dst_ports`
                - Type: int
            - Field: `cid`
                - Type int

A new `http` subdocument is stored in the `dat` array during each import session.

The IP addresses of the HTTP servers seen responding to the HTTP server name are stored in the `http.dst_ips` array.

The ports on which the HTTP servers responded to the HTTP server name are stored in the `http.dst_ports` array.

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations.

### HTTP Strobe Designation
Inputs:
- `Config.S.Strobe.ConnectionLimit`
    - Type: int
- `ParseResults.HTTPConnMap` created by `FSImporter`
    - Field: `ConnectionCount`
        - Type: int

Outputs:
- MongoDB `SNIconn` collection:
    - Array Field: `dat`
        - Object Field: `http`
            - Field: `strobe`
                - Type: bool

This field is included in same `dat.http` subdocument as the destination IP addresses described above.

If the number of TLS connections from the source to the destination in the set of network logs under consideration is greater than the strobe connection limit, the SNI connection is marked as a strobe. These hosts can be considered to have been in constant communication.

### HTTP Connection Statistics
Inputs:
- `ParseResults.HTTPConnMap` created by `FSImporter`
    - Field: `ConnectionCount`
        - Type: int
    - Field: `ZeekUIDs`
        - Type: []string
- `ParseResults.ZeekUIDMap` created by `FSImporter`
    - Struct Field: `Conn`
        - Field: `OrigBytes`
            - Type: int64
        - Field: `RespBytes`
            - Type: int64
        - Field: `Duration`
            - Type: float64

Outputs:
- MongoDB `SNIconn` collection:
    - Array Field: `dat`
        - Object Field: `http`
            - Field: `count`
                - Type: int
            - Field: `tbytes`
                - Type: int
            - Field: `tdur`
                - Type: float64

These fields are included in same `dat.http` subdocument as the destination IP addresses described above.

The number of connections from the from the source IP address to the destination HTTP server name is stored in the `count` field.

The total number of bytes sent from the source to the destination is summed together with the number of bytes sent back to the source from the destination and stored in the `tbytes` field.

The total duration of the connection from the source to the destination is stored in the `tdur` field. These duration fields are used to support long connection analysis.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the total connection count, total bytes, all of the `dat.http` subdocuments must be summed together.


### HTTP Connection Timestamps and Originating Bytes
Inputs:
- `ParseResults.HTTPConnMap` created by `FSImporter`
    - Field: `Timestamps`
        - Type: data.IntSet
    - Field: `ZeekUIDs`
        - Type: []string
- `ParseResults.ZeekUIDMap` created by `FSImporter`
    - Struct Field: `Conn`
        - Field: `OrigBytes`
            - Type: int64

Outputs:
- MongoDB `SNIconn` collection:
    - Array Field: `dat`
        - Object Field: `http`
            - Array Field: `bytes`
                - Type: int
            - Array Field: `ts`
                - Type: int

These fields are included in same `dat.http` subdocument as the destination IP addresses described above.

The individual timestamps of the connections from the source to the destination are unioned together and stored in MongoDB. Additionally, the number of bytes the source sent to the destination in each of the connections is stored. The `beaconSNI` package takes these outputs as input.

In order to gather all of the connection timestamps across chunked imports, the `ts` arrays from each of the `dat.http` documents must be unioned together. Similarly, in order to gather all of the data sizes across chunked imports, the `bytes` arrays from each of the `dat.http` subdocuments must be concatenated.

If a connection is marked as a strobe, these fields may be missing or empty.

### HTTP Connection Details
Inputs:
- `ParseResults.HTTPConnMap` created by `FSImporter`
    - Field: `Methods`
        - Type: data.StringSet
    - Field: `UserAgents`
        - Type: data.StringSet

Outputs:
- MongoDB `SNIconn` collection:
    - Array Field: `dat`
        - Object Field: `http`
            - Array Field: `methods`
                - Type: string
            - Array Field: `user_agents`
                - Type: string

These fields are included in same `dat.http` subdocument as the destination IP addresses described above.

The HTTP methods sent in the HTTP connections from the source to the destination are stored in the `methods` field.

The HTTP user agents sent in the HTTP connections from the source to the destination are stored in the `user_agents` field.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the whole set of HTTP methods or user agents, these arrays in the `http` subdocuments must be unioned together.

### SNI Beacon Strobe Designation
Inputs:
- `Config.S.Strobe.ConnectionLimit`
    - Type: int
- MongoDB `SNIconn` collection:
    - Array Field: `dat`
        - Object Field: `tls`
            - Field: `count`
                - Type: int
        - Object Field: `http`
            - Field: `count`
                - Type: int

Outputs:
- MongoDB `SNIconn` collection:
    - Array Field: `dat`
        - Object Field: `beacon`
            - Field: `strobe`
                - Type: bool
            - Field: `cid`
                - Type: int

During the beaconSNI connection analysis, the connections over HTTP and TLS are gathered together. If the total number of connections exceeds the strobe limit, the beaconSNI package will insert a new subdocument into the pair's `SNIconn` record. 

This document is only created if neither `dat.tls.strobe` nor `dat.http.strobe` have been set to true. As a result, the following fields must all be queried when searching for SNI connection strobes:
- `dat.tls.strobe`
- `dat.http.strobe`
- `dat.beacon.strobe`
