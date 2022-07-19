## Hostname Package 

*Documented on May 24, 2022*

---

This package records the hostnames/ fully qualified domain names (FQDN) seen in the current set of network logs under consideration. DNS query logs are used to gather FQDNs and the IP addresses they resolve to. 

This package records the following:
- Fully qualified domain names
- Whether the FQDN appears on any threat intel lists
- The IP addresses the FQDN was seen resolving to
- The IP addresses which made DNS queries for the FQDN

## Package Outputs

### Fully Qualified Domain Name

Inputs: 
- `ParseResults.HostnameMap` created by `FSImporter`
    - Field: `Host`
        - Type: string

Outputs:
- MongoDB `hostname` collection:
    - Field: `host`
        - Type: string

The `host` field of each document in the `hostnames` collection records the FQDN of a host that was queried for in the DNS logs currently under consideration.

### Chunk ID
Inputs: 
- `Config.S.Rolling.CurrentChunk`
    - Type: int

Outputs:
- MongoDB `host` collection:
    - Field: `cid`
        - Type: int

The `cid` field records the chunk ID of the import session in which this host document was last updated. This field is used to support rolling imports.

### Threat Intel Designation
Inputs:
- MongoDB `hostname` collection in the `rita-bl` database
    - Field: `index`
        - Type: string
- `ParseResults.HostnameMap` created by `FSImporter`
    - Field: `Host`
        - Type: string

Outputs:
- MongoDB `hostname` collection:
    - Field: `blacklisted`
        - Type: bool

This field marks whether the FQDN has appeared on any threat intelligence lists managed by `rita-bl`. These lists are registered in the RITA configuration file.

### Query Originator and Resolved IP Addresses 
- `ParseResults.HostnameMap` created by `FSImporter`
    - Field: `ClientIPs`
        - Type: data.UniqueIPSet
    - Field: `ResolvedIPs`
        - Type: data.UniqueIPSet

Outputs:
- MongoDB `hostname` collection:
    - Array Field: `dat`
        - Array Field: `ips`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
            - Field: `network_name`
                - Type: string
        - Array Field: `src_ips`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
            - Field: `network_name`
                - Type: string
        - Field: `cid`
            - Type: int

The set of IP addresses which queried for a given FQDN is stored in `dat.src_ips` as an array of Unique IP addresses. Similarly, the set of IP addresses which the FQDN was seen to resolve to are stored in `dat.ips` as an array of Unique IP addresses. 

In order to gather all of the query originator IP addresses or resolved IP addresses for an FQDN across chunked imports, the `src_ips` or `ips` arrays from each of the `dat` documents must be unioned together. 