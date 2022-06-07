## Certificate Package

*Documented on June 7, 2022*

---

This package records the IP addresses of servers which presented invalid TLS certificates in the current set of network logs under consideration.

This package records the following:
- TLS server IP addresses
- The client IP addresses which connected to the TLS server and were presented invalid certificates
- The reasons why the certificate presented by the server is invalid
- How many times the server presented an invalid certificate

## Package Outputs

### Server Unique IP Address

Inputs:
- `ParseResults.CertificateMap` created by `FSImporter`
    - Field: `Host`
        - Type: data.UniqueIP
Outputs:
- MongoDB `cert` collection:
    - Field: `ip`
        - Type: string
    - Field: `network_uuid`
        - Type: UUID
    - Field: `network_name`
        - Type: string

The servers which present invalid certificates in the network logs under consideration are stored as unique IP addresses in the `cert` collection. 

The `ip` field records the string representation of the IP address. The `network_uuid` and `network_name` fields have been introduced to disambiguate hosts using the same private IP address on separate networks. 

These fields are used to select an individual entry in the `cert` collection. All of the other outputs described here use the `ip` and `network_uuid` fields as selectors when updating `cert` collection entries in MongoDB.

### Chunk ID
Inputs: 
- `Config.S.Rolling.CurrentChunk`
    - Type: int

Outputs:
- MongoDB `host` collection:
    - Field: `cid`
        - Type: int

The `cid` field records the chunk ID of the import session in which this host document was last updated. This field is used to support rolling imports.

### Source Unique IP Addresses
Inputs:
- `ParseResults.CertificateMap` created by `FSImporter`
    - Field: `OrigIps`
        - Type: data.UniqueIPSet

Outputs:
- MongoDB `cert` collection:
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

The IP addresses seen sending connections to the server presenting an invalid certificate are stored in the `orig_ips` array of a new `dat` subdocument during each import session.

As an implementation detail, the `orig_ips` array is truncated to 200003 IP addresses in each subdocument.

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations.

### Port, Protocol, Service Triplets
Inputs: 
- `ParseResults.CertificateMap` created by `FSImporter`
    - Field: `Tuples`
        - Type: data.StringSet

Outputs:
- MongoDB `cert` collection:
    - Array Field: `dat`
        - Array Field: `tuples`
            - Type: string

This field is included in same `dat` subdocument as the source unique IP addresses.

The destination port, transport protocol (icmp, udp, or tcp), and service protocol (e.g. ssh) of each connection in which the server presented an invalid certificate are grouped together to form a triplet. The set of these triplets is then stored in MongoDB. 

In order to gather all of the triplets across chunked imports, the `tuples` arrays from each the `dat` subdocuments must be unioned together. 

### Certificate Validation Errors
Inputs:
- `ParseResults.CertificateMap` created by `FSImporter`
    - Field: `InvalidCerts`
        - Type: data.StringSet

Outputs:
- MongoDB `cert` collection:
    - Array Field: `dat`
        - Field: `icodes`
            - Type: string

The set of certificate validation errors associated with the invalid certificate presented by the server are stored in the `icodes` array in a `dat` subdocument during each import session.

This field is included in same `dat` subdocument as the source unique IP addresses.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return all of the validation errors associated with certificates presented by the server, the union of the `dat` subdocuments must be taken.


### Certificate Frequency
Inputs:
- `ParseResults.CertificateMap` created by `FSImporter`
    - Field: `Seen`
        - Type: int64

Outputs:
- MongoDB `cert` collection:
    - Array Field: `dat`
        - Array Field: `seen`
            - Type: int64

The number of times the server presented an invalid certificate is stored in the `seen` field in a `dat` subdocument during the each import session.

This field is included in same `dat` subdocument as the source unique IP addresses.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the total count of how many times the server presented an invalid certificate, the sum of the `dat` subdocuments must be taken.