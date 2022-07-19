## Useragent Package

*Documented on June 2, 2022*

---

This package records connection signatures such as HTTP useragents and JA3 hashes. Rare signatures often point to interesting communications on the network. 

This package records the following:
- Connection signatures
- How often each signature was seen on the network
- The IP addresses which made connections with the signature
- The FQDNs which received connections with the signature

## Package Outputs 

### Connection Signature
Inputs:
- `ParseResults.UseragentMap` created by `FSImporter`
    - Field: `Name`
        - Type: string

Outputs:
- MongoDB `useragent` collection:
    - Field: `user_agent`
        - Type: string

Connection signatures are stored in the `user_agent` field. RITA currently supports recording HTTP useragent strings and JA3 hashes as connection signatures. Both types of signatures are stored in the same collection using the same fields.

The `user_agent` field may be truncated to the first 800 characters if the signature is too long to be indexed with MongoDB.

This field is used to select an individual entry in the `useragent` collection. All of the other outputs described here use the `user_agent` field as selectors when updating `useragent` collection entries in MongoDB.

### JA3 Field
Inputs:
- `ParseResults.UseragentMap` created by `FSImporter`
    - Field: `JA3`
        - Type: bool

Outputs:
- MongoDB `useragent` collection:
    - Field: `ja3`
        - Type: bool

The `ja3` boolean field was introduced, in order to disambiguate HTTP useragents from JA3 signatures.

### Chunk ID
Inputs: 
- `Config.S.Rolling.CurrentChunk`
    - Type: int

Outputs:
- MongoDB `useragent` collection:
    - Field: `cid`
        - Type: int

The `cid` field records the chunk ID of the import session in which this host document was last updated. This field is used to support rolling imports.

### Source Unique IP Addresses
Inputs:
- `ParseResults.UseragentMap` created by `FSImporter`
    - Field: `OrigIps`
        - Type: data.UniqueIPSet

Outputs:
- MongoDB `useragent` collection:
    - Array Field: `dat`
        - Array Field: `orig_ips`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
            - Field: `network_name`
                - Type: string
        - Field: `cid`
            - Type: int

The IP addresses seen sending connections with the given signature are stored in the `orig_ips` array of a new `dat` subdocument during each import session.

As an implementation detail, the `orig_ips` array is truncated to 10 IP addresses in each subdocument.

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations.

### Destination FQDNs
Inputs:
- `ParseResults.UseragentMap` created by `FSImporter`
    - Field: `Requests`
        - Type: data.StringSet

Outputs:
- MongoDB `useragent` collection:
    - Array Field: `dat`
        - Array Field: `hosts`
            - Type: string

The servers seen responding to connections with the given signature are stored in the `hosts` array of a `dat` subdocument during each import session. The server identifiers are pulled from the Host header for HTTP requests and the TLS server name indicator for SSL connections.

As an implementation detail, the `hosts` array is truncated to 10 IP addresses in each subdocument.

This field is included in same `dat` subdocument as the source unique IP addresses.

### Signature Frequency
Inputs:
- `ParseResults.UseragentMap` created by `FSImporter`
    - Field: `Seen`
        - Type: int64

Outputs:
- MongoDB `useragent` collection:
    - Array Field: `dat`
        - Array Field: `seen`
            - Type: int64

The number of times the signature was seen is stored in the `seen` field in a `dat` subdocument during the each import session.

This field is included in same `dat` subdocument as the source unique IP addresses.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the total count of how many times the signature appeared, the sum of the `dat` subdocument must be taken.

### Rare Signature Summary

Inputs:
- MongoDB `useragent` collection:
    - Array Field: `dat`
        - Array Field: `orig_ips`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
            - Field: `network_name`
                - Type: string

Outputs:
- MongoDB `host` collection:
    - Array Field: `dat`
        - Field: `rsig`
            - Type: string
        - Field: `rsigc`
            - Type: int
        - Field: `cid`
            - Type: int

After the main signature analysis, signatures associated with less than 5 originating hosts are recorded in the `host` collection.

A new subdocument is created in each of the originating hosts' `dat` arrays. The `rsig` field records the rare signature. The `rsigc` field is always set to 1.

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

There should always be one `dat` subdocument per rare signature associated with this host. There should not be multiple `dat` subdocuments with the same `rsig` field.