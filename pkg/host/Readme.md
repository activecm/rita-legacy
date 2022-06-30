## Host Package

*Documented on May 17, 2022*

---

This package records the hosts seen in the current set of network logs under consideration. 

Initially, this package only recorded:
- The IP address of a host
- Whether a host is IPv4 or IPv6
- The binary representation of the IP address
- Whether a host is internal (local) or external (public)
- How many times the host appears as the source of a connection
- How many times the host appears as the destination of a connection

Over time, this package has come to perform several other calculations that could be moved to other packages. This includes:
- How many times the host appears as the source of a connection with a well-known port/ protocol mismatch
- Whether the host appears on any threat intel lists
- An exploded DNS analysis of all the DNS queries made by the host

## Package Outputs

### Unique IP Address

Inputs: 
- `ParseResults.HostMap` created by `FSImporter`
    - Field: `Host`
        - Type: data.UniqueIP

Outputs:
- MongoDB `host` collection:
    - Field: `ip`
        - Type: string
    - Field: `network_uuid`
        - Type: UUID
    - Field: `network_name`
        - Type: string

The `ip` field records the string representation of an IP address seen in the network logs. The `network_uuid` and `network_name` fields have been introduced to disambiguate hosts using the same private IP address on separate networks. 

These fields are used to select an individual entry in the `host` collection. All of the other outputs described here use the `ip` and `network_uuid` fields as selectors when updating `host` collection entries in MongoDB.

### Chunk ID
Inputs: 
- `Config.S.Rolling.CurrentChunk`
    - Type: int

Outputs:
- MongoDB `host` collection:
    - Field: `cid`
        - Type: int

The `cid` field records the chunk ID of the import session in which this host document was last updated. This field is used to support rolling imports.

### Local/ Internal Designation
Inputs: 
- `ParseResults.HostMap` created by `FSImporter`
    - Field: `IsLocal`
        - Type: bool

Outputs:
- MongoDB `host` collection:
    - Field: `local`
        - Type: bool

The `local` field records whether a host is considered internal or external to the observation environment. Certain analyses are only performed on connections from internal hosts to external hosts. The importer determines whether a host is local or not using the networks defined in the RITA configuration file.

### IPv4 Binary Representation
Inputs: 
- `ParseResults.HostMap` created by `FSImporter`
    - Field: `IP4`
        - Type: bool
    - Field: `IP4Bin`
        - Type: int64

Outputs:
- MongoDB `host` collection:
    - Field: `ipv4`
        - Type: bool
    - Field: `ipv4_binary`
        - Type: int

When querying network hosts using CIDR notation, it is helpful to have access to the binary representations of IP addresses. IPv4 addresses are stored as 64 bit integers in MongoDB in order to support this use case. However, MongoDB lacks a datatype for storing and querying binary IPv6 addresses.

### Threat Intel Designation
Inputs:
- MongoDB `ip` collection in the `rita-bl` database
    - Field: `index`
        - Type: string
- `ParseResults.HostMap` created by `FSImporter`
    - Field: `Host`
        - Type: data.UniqueIP

Outputs:
- MongoDB `host` collection:
    - Field: `blacklisted`
        - Type: bool

This field marks whether the IP address has appeared on any threat intelligence lists managed by `rita-bl`. These lists are registered in the RITA configuration file.

### Connection Counts
Inputs: 
- `ParseResults.HostMap` created by `FSImporter`
    - Field: `CountSrc`
        - Type: int
    - Field: `CountDst`
        - Type: int
    - Field: `UntrustedAppConnCount`
        - Type: int
- `Config.S.Rolling.CurrentChunk`
    - Type: int

Outputs:
- MongoDB `host` collection:
    - Array Field: `dat`
        - Field: `count_src`
            - Type: int
        - Field: `count_dst`
            - Type: int
        - Field: `upps_count`
            - Type: int
        - Field: `cid`
            - Type: int

The connection count analysis pushes a new subdocument in the the `host` entry's `dat` array. This subdocument details how many times the host appeared as a source or destination in the logs being currently processed. Additionally, this analysis records how often the host appeared as the source of a connection with a well-known port/ protocol mismatch. 

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the total connection counts, all of the subdocuments must be summed together.

### Host Scoped Exploded DNS Analysis
Inputs: 
- `ParseResults.HostMap` created by `FSImporter`
    - Field: `DNSQueryCount`
        - Type: map[string]int64
- `Config.S.Rolling.CurrentChunk`
    - Type: int

Outputs:
- MongoDB `host` collection:
    - Array Field: `dat`
        - Array Field: `exploded_dns`
            - Field: `query`
                - Type: string
            - Field: `count`
                - Type: int
        - Field: `cid`
            - Type: int

The host package performs an exploded DNS analysis for the DNS queries made by each individual host. See the `explodeddns` package for more information. The `query` fields contain the super-domains that were queried for by this host, and the associated `count` fields contain how often the super-domains were queried.

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the total query counts, all of the subdocuments must be summed together.

### Exploded DNS Analysis Summary: Maximum Queries
Inputs:
- `ParseResults.HostMap` created by `FSImporter`
    - Field: `Host`
        - Type: data.UniqueIP
    - Field: `IsLocal`
        - Type: bool
- MongoDB `host` collection:
    - Array Field: `dat`
        - Array Field: `exploded_dns`
            - Field: `query`
                - Type: string
            - Field: `count`
                - Type: int64

Outputs:
- MongoDB `host` collection:
    - Array Field: `dat`
        - Field: `max_dns.query`
            - Type: string
        - Field: `max_dns.count`
            - Type: int
        - Field: `cid`
            - Type: int

After running the initial analysis, a summary is produced for each local host which finds the super-domain with the most queries *over the entire observation period*. 

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the super-domain with the most queries, the maximum of the these subdocuments must be taken.
