## Unique Connections Package

*Documented on May 23, 2022*

---
This package records the details of the connections made between pairs of hosts in the set of network logs under consideration. RITA refers to the set of connections from one host to another using the term "unique connection" or "uconn" for short. 

This package records the following:
- The pair of IP addresses that communicated
- How many times the source IP address connected to the destination IP address
    - Unique connections with connection counts exceeding the limit defined in the RITA configuration are marked as "strobes"
- The longest connection from the source to the destination
- The total amount of time the source was connected to the destination
- The total amount of bytes sent between the pair
- Timestamps of the individual connections
- The amount of data sent from the source to the destination in each of the individual connections
- The (destination port, protocol, service) combinations that the client used when communicating with the destination
- Open connection tracking of some of the above data

## Package Outputs

### Source and Destination Unique IP Address Pair
Inputs: 
- `ParseResults.UniqueConnMap` created by `FSImporter`
    - Field: `Hosts`
        - Type: data.UniqueIPPair

Outputs:
- MongoDB `uconn` collection:
    - Field: `src`
        - Type: string
    - Field: `src_network_uuid`
        - Type: UUID
    - Field: `src_network_name`
        - Type: string
    - Field: `dst`
        - Type: string
    - Field: `dst_network_uuid`
        - Type: UUID
    - Field: `dst_network_name`
        - Type: string

The `src` and `dst` fields record the string representation of the source and destination IP addresses as seen in the network logs. The `src_network_uuid`, `src_network_name`, `dst_network_uuid`, and `dst_network_name` fields have been introduced to disambiguate hosts using the same private IP address on separate networks.

These fields are used to select an individual entry in the `uconn` collection. All of the other outputs described here use the `src`, `src_network_uuid`, `dst`, and `dst_network_uuid` fields as selectors when updating `uconn` collection entries in MongoDB.

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
    - Type: int
- `ParseResults.UniqueConnMap` created by `FSImporter`
    - Field: `ConnectionCount`
        - Type: int

Outputs:
- MongoDB `uconn` collection:
    - Field: `strobe`
        - Type: bool

If the number of connections from the source to the destination in the set of network logs under consideration is greater than the strobe connection limit, the unique connection is marked as a strobe. These hosts can be considered to have been in constant communication.

Unique connections may become strobes over time due to chunked imports. The `beacon` package handles updating this field when a unique connection breaks over the strobe limit due to these chunked imports.

### Unique Connection Statistics
Inputs: 
- `ParseResults.UniqueConnMap` created by `FSImporter`
    - Field: `ConnectionCount`
        - Type: int
    - Field: `TotalBytes`
        - Type: int
    - Field: `TotalDuration`
        - Type: float64
    - Field: `MaxDuration`
        - Type: float64

Outputs:
- MongoDB `uconn` collection:
    - Array Field: `dat`
        - Field: `count`
            - Type: int
        - Field: `tbytes`
            - Type: int
        - Field: `maxdur`
            - Type: float64
        - Field: `tdur`
            - Type: float64
        - Field: `cid`
            - Type: int

The number of connections from the source to the destination in the network logs under consideration are stored in the `dat.count` field. 

The total number of bytes sent from the source to the destination is summed together with the number of bytes sent back to the source from the destination and stored in the `tbytes` field.

The length of the longest connection  from the source to the destination is stored in the `dat.maxdur` field in seconds. The total duration of the connection from the source to the destination is stored in the `tdur` field. These duration fields are used to support long connection analysis.

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the total connection count, total bytes, or total duration, all of the subdocuments must be summed together. In order to return the longest connection duration, the maximum of the subdocuments must be taken.

### Connection Timestamps and Originating Bytes
Inputs: 
- `ParseResults.UniqueConnMap` created by `FSImporter`
    - Field: `TsList`
        - Type: []int64
    - Field: `OrigBytesList`
        - Type: []int64

Outputs:
- MongoDB `uconn` collection:
    - Array Field: `dat`
        - Array Field: `bytes`
            - Type: int
        - Array Field: `ts`
            - Type: int

These fields are stored in the same subdocument as the unique connection statistics above. 

The individual timestamps of the connections from the source to the destination are unioned together and stored in MongoDB. Additionally, the number of bytes the source sent to the destination in each of the connections is stored. The `beacon` package takes these outputs as input.

In order to gather all of the connection timestamps across chunked imports, the `ts` arrays from each of the `dat` documents must be unioned together. Similarly, in order to gather all of the data sizes across chunked imports, the `bytes` arrays from each of the `dat` subdocuments must be concatenated. 

If a connection is marked as a strobe, these fields may be missing or empty.

### Port, Protocol, Service Triplets
Inputs: 
- `ParseResults.UniqueConnMap` created by `FSImporter`
    - Field: `Tuples`
        - Type: data.StringSet

Outputs:
- MongoDB `uconn` collection:
    - Array Field: `dat`
        - Array Field: `tuples`
            - Type: string

These fields are stored in the same subdocument as the unique connection statistics above. 

The destination port, transport protocol (icmp, udp, or tcp), and service protocol (e.g. ssh) of each individual connection are grouped together to form a triplet. The set of these triplets seen for each unique connection is then stored in MongoDB. 

In order to gather all of the triplets across chunked imports, the `tuples` arrays from each the `dat` subdocuments must be unioned together. 

### Invalid Certificate Designation
Inputs: 
- `ParseResults.UniqueConnMap` created by `FSImporter`
    - Field: `InvalidCertFlag`
        - Type: bool

Outputs:
- MongoDB `uconn` collection:
    - Array Field: `dat`
        - Array Field: `icerts`
            - Type: bool


These fields are stored in the same subdocument as the unique connection statistics above. 

If an invalid certificate was presented by the destination of a unique connection, we set the `dat.icerts` flag to true.

### Open Connection Tracking
Inputs: 
- `ParseResults.UniqueConnMap` created by `FSImporter`
    - Collection: `.ConnStateMap`
        - Field: `Bytes`
            - Type: int64
        - Field: `Duration`
            - Type: float64
        - Field: `OrigBytes`
            - Type: int64
        - Field: `Ts`
            - Type: int64

Outputs:
- MongoDB `uconn` collection:
    - Field: `open`
        - Type: bool
    - Field: `open_bytes`
        - Type: int
    - Field: `open_connection_count`
        - Type: int
    - Field: `open_duration`
        - Type: float64
    - Field: `open_orig_bytes`
        - Type int
    - Array Field: `open_ts`:
        - Type int
    - Field: `open_conns`
        - Type: object
        - Values as Array Field:
            - Field: `bytes`
                - Type: int
            - Field: `duration`
                - Type: float64
            - Field: `open`
                - Type: bool
            - Field: `orig_bytes`
                - Type: int
            - Field: `ts`
                - Type: int
            - Field: `tuple`
                - Type: string

When the [open connections Zeek package](https://github.com/activecm/zeek-open-connections) is loaded, RITA tracks open connections alongside the closed connections which make up each unique connection.

The `open` field is set to true when at least one open connection exists between the source and destination IP addresses.

The metrics tracked for open connections are similar to those that are tracked in the Unique Connections Statistics and Connection Timestamps and Originating Bytes sections.


| Open Connections Field | Closed Connections Field |
|------------------------|--------------------------|
|`.open_bytes`           |`.dat.tbytes`             |
|`.open_connection_count`|`.dat.count`              |
|`.open_duration`        |`.dat.tdur`               |
|`.open_orig_bytes`      |sum `.dat.bytes`          |
|`.open_ts`              |`.dat.ts`                 |

The individual open connections are stored in the `open_conns` field. Due to implementation details, the `open_conns` field is stored as an object rather than array. The MongoDB operator `$objectToArray` is helpful when aggregating open connections. The keys of the `open_conns` object should not be depended upon. Only the values of the `open_conns` object should be read.

| Open Connection Field  | Closed Connections Field |
|------------------------|--------------------------|
|`.bytes`                |`.dat.tbytes`             |
|`.duration`             |`.dat.tdur`               |
|`.orig_bytes`           |index `.dat.bytes`        |
|`.ts`                   |index `.dat.ts`           |

The `open` filed of each individual open connection should always be set to `true`.

### Host Fetched an Invalid Certificate Summary

Inputs: 
- `ParseResults.HostMap` created by `FSImporter`
    - Field: `IsLocal`
        - Type: bool
- MongoDB `uconn` collection:
    - Array Field: `dat`
        - Field: `icerts`
            - Type: bool
        - Field: `cid`
            - Type: int

Outputs:
- MongoDB `host` collection:
    - Array Field: `dat`
        - Object Field: `icdst`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
            - Field: `network_name`
                - Type: string
        - Field: `icert`
            - Type: int
        - Field: `cid`
            - Type: int

After building the unique connections collection, RITA checks if any of the internal hosts seen in the logs under consideration was the source of a connection associated with an invalid certificate. The `host` record's `dat.icdst` field records the server which presented that host an invalid certificate. The `host` record's `dat.icert` field is always set to 1.

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

### Longest Connection Summary

Inputs: 
- `ParseResults.HostMap` created by `FSImporter`
    - Field: `IsLocal`
        - Type: bool
    - Field: `Host`
        - Type: data.UniqueIP
- MongoDB `uconn` collection:
    - Field: `src`
        - Type: string
    - Field: `src_network_uuid`
        - Type: UUID
    - Field: `src_network_name`
        - Type: string
    - Field: `dst`
        - Type: string
    - Field: `dst_network_uuid`
        - Type: UUID
    - Field: `dst_network_name`
        - Type: string
    - Array Field: `dat`
        - Field: `maxdur`
            - Type: float64
        - Field: `cid`
            - Type: int

Outputs:
- MongoDB `host` collection:
    - Array Field: `dat`
        - Object Field: `mdip`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
            - Field: `network_name`
                - Type: string
        - Field: `max_duration`
            - Type: int
        - Field: `cid`
            - Type: int

After building the unique connections collection, RITA finds the external hosts which spent the most amount of time connected to each of the internal hosts.

The `host` record's `dat.mdip` field stores the external IP address of the unique connection with the highest `dat.maxdur` in which the internal host took part. The `dat.max_duration` field stores the associated `dat.maxdur` value. This analysis only considers unique connections updated in the current chunk.

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the longest connection made by an internal host, the maximum of the these subdocuments must be taken.