## Unique Proxy Connections Package

*Documented on May 23, 2022*

---
This package records the details of connections made between IP addresses and fully qualified domain names (FQDN) using HTTP proxies. RITA refers to the set of connections from one host to an fqdn using the term "unique connection" or "uconn" for short. 

This package records the following:
- The source IP address and destination FQDN of proxied connections
- The IP address of the the last used proxy between a source IP address and a destination FQDN
- How many times the source IP address connected to the destination FQDN via a HTTP proxy
    - Unique connections with connection counts exceeding the limit defined in the RITA configuration are marked as "strobes"
- Timestamps of the individual proxied connections

## Package Outputs

### Source Unique IP, Destination FQDN Pair
Inputs:
- `ParseResults.ProxyUniqueConnMap` created by `FSImporter`
    - Field: `Hosts`
        - Type: data.UniqueSrcFQDNPair
Outputs:
- MongoDB `uconnProxy` collection:
    - Field: `src`
        - Type: string
    - Field: `src_network_uuid`
        - Type: UUID
    - Field: `src_network_name`
        - Type: string
    - Field: `fqdn`
        - Type: string

The `src` field records the string representation of the IP address of the source of a proxied connection as seen in the network logs. Similarly, the field `fqdn` specifies the fully qualified domain name of the destination of a proxied connection as seen in the network logs. The `src_network_uuid` and `src_network` name fields have been introduced to disambiguate hosts using the same private IP address on separate networks. 

These fields are used to select an individual entry in the `uconnProxy` collection. All of the other outputs described here use the `src`, `src_network_uuid`, and `fqdn` fields as selectors when updating `uconnProxy` collection entries in MongoDB.

### Chunk ID
Inputs: 
- `Config.S.Rolling.CurrentChunk`
    - Type: int

Outputs:
- MongoDB `uconn` collection:
    - Field: `cid`
        - Type: int

The `cid` field records the chunk ID of the import session in which this unique connection document was last updated. This field is used to support rolling imports.

### Strobe Designation
Inputs: 
- `Config.S.Strobe.ConnectionLimit`
    - Type: int64
- `ParseResults.ProxyUniqueConnMap` created by `FSImporter`
    - Field: `ConnectionCount`
        - Type: int64

Outputs:
- MongoDB `uconnProxy` collection:
    - Field: `strobeFQDN`
        - Type: bool

If the number of connections from the source to the destination in the set of network logs under consideration is greater than the strobe connection limit, the unique connection is marked as a strobe. These hosts can be considered to have been in constant communication.

Unique connections may become strobes over time due to chunked imports. The `beaconProxy` package handles updating this field when a unique connection breaks over the strobe limit due to these chunked imports.

### Last Seen Proxy Server
Inputs:
- `ParseResults.ProxyUniqueConnMap` created by `FSImporter`
    - Field: `Proxy`
        - Type: data.UniqueIP

Outputs:
- MongoDB `uconnProxy` collection:
    - Object Field: `proxy`
        - Field: `ip`
            - Type: string
        - Field: `network_uuid`
            - Type: UUID
        - Field: `network_name`
            - Type: string

The IP address of the last proxy server which serviced a request from the source IP to connect to the destination FQDN is stored in the `proxy` field.

### Proxied Unique Connection Statistics
Inputs: 
- `ParseResults.ProxyUniqueConnMap` created by `FSImporter`
    - Field: `ConnectionCount`
        - Type: int64

Outputs:
- MongoDB `uconnProxy` collection:
    - Array Field: `dat`
        - Field: `count`
            - Type: int64

The number of connections from the source to the destination in the network logs under consideration are stored in the `dat.count` field. 

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the total connection count, all of the subdocuments must be summed together. 

### Connection Timestamps
Inputs:
- `ParseResults.ProxyUniqueConnMap` created by `FSImporter`
    - Field: `TsList`
        - Type: []int64

Outputs:
- MongoDB `uconnProxy` collection:
    - Array Field: `dat`
        - Array Field: `ts`
            - Type: int64

These fields are stored in the same subdocument as the unique connection statistics above. 

The individual timestamps of the connections from the source to the destination are unioned together and stored in MongoDB. 

In order to gather all of the connection timestamps across chunked imports, the `ts` arrays from each of the `dat` documents must be unioned together. 

If a connection is marked as a strobe, these fields may be missing or empty.
