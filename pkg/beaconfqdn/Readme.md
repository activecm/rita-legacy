## FQDN Beacon Package

*Documented on May 31, 2022*

---
This package analyzes connections between internal hosts and fully qualified domain names (FQDNs) for signs of regular, programmatic communication.

This package records the following:
- The IP address, FQDN pair that communicated
- Summary statistics of the connections between the pair
    - Pairs with connection counts exceeding the limit defined in the RITA configuration are marked as "strobes"
- Timestamp beaconing statistics
- Data size beaconing statistics
- Beacon scoring results

## Package Outputs

### Source Unique IP, Destination FQDN Pair
Inputs:
- `ParseResults.HostnameMap` created by `FSImporter`
    - Field: `Host`
        - Type: string
- MongoDB `hostname` collection:
    - Field: `host`
        - Type: string
    - Array Field: `dat`
        - Array Field: `ips`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
            - Field: `network_name`
                - Type: string
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

Outputs:
- MongoDB `beaconFQDN` collection:
    - Field: `src`
        - Type: string
    - Field: `src_network_uuid`
        - Type: UUID
    - Field: `src_network_name`
        - Type: string
    - Field: `fqdn`
        - Type: string
    - Array Field: `resolved_ips`
        - Type: string


After the `hostname` package runs, the `beaconfqdn` package gathers the unique connections to each FQDN from the network logs under consideration. First, the FQDN is resolved to the set of underlying IP addresses using the `hostname` collection. Then, the unique connections in which the IP addresses are the destinations are gathered from the `uconn` collection. Finally, these unique connections are grouped by their source IP addresses. 

The `src` field records the string representation of the source IP address of the set of unique connections to the destination IP addresses associated with the fully qualified domain name stored in the `fqdn` field.  The `src_network_uuid` and `src_network` name fields have been introduced to disambiguate hosts using the same private IP address on separate networks. 

These fields are used to select an individual entry in the `beaconFQDN` collection. All of the other outputs described here use the `src`, `src_network_uuid`, and `fqdn` fields as selectors when updating `beaconFQDN` collection entries in MongoDB.

The `resolved_ips` field stores the set of IP addresses the FQDN resolved to during the `beaconFQDN` analysis.

### Chunk ID
Inputs: 
- `Config.S.Rolling.CurrentChunk`
    - Type: int

Outputs:
- MongoDB `uconn` collection:
    - Field: `cid`
        - Type: int

The `cid` field records the chunk ID of the import session in which this unique connection document was last updated. This field is used to support rolling imports.

### Unique Connection Summary Statistics
Inputs:
- `ParseResults.HostnameMap` created by `FSImporter`
    - Field: `Host`
        - Type: string
- MongoDB `hostname` collection:
    - Field: `host`
        - Type: string
    - Array Field: `dat`
        - Array Field: `ips`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
- MongoDB `uconn` collection:
    - Field: `src`
        - Type: string
    - Field: `src_network_uuid`
        - Type: UUID
    - Field: `dst`
        - Type: string
    - Field: `dst_network_uuid`
        - Type: UUID
    - Array Field: `dat`
        - Array Field: `bytes`
            - Type: int
        - Field: `count`
            - Type: int

Outputs:
- MongoDB `beaconFQDN` collection:
    - Field: `connection_count`
        - Type: int
    - Field: `avg_bytes`
        - Type: float64

First, a set of (IP address, IP address) unique connections is gathered for the (IP address, FQDN) pair under consideration.

The `dat.count` fields from the pair's corresponding `uconn` documents are summed together in order to find the total amount of connections from the source IP address to the destination IP. The result is stored in the `connection_count` field of the pair's `beaconFQDN` document.

The `dat.bytes` arrays from the `uconn` documents are concatenated and the average of the values stored in the `avg_bytes` field of the pair's `beaconFQDN` document. 

### Strobe Designation
Inputs:
- `ParseResults.HostnameMap` created by `FSImporter`
    - Field: `Host`
        - Type: string
- MongoDB `hostname` collection:
    - Field: `host`
        - Type: string
    - Array Field: `dat`
        - Array Field: `ips`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
- MongoDB `uconn` collection:
    - Field: `src`
        - Type: string
    - Field: `src_network_uuid`
        - Type: UUID
    - Field: `dst`
        - Type: string
    - Field: `dst_network_uuid`
        - Type: UUID
    - Array Field: `dat`
        - Array Field: `bytes`
            - Type: int
        - Field: `count`
            - Type: int

Outputs:
- MongoDB `beaconFQDN` collection:
    - Field: `strobeFQDN`
        - Type: boolean
        
First, a set of (IP address, IP address) unique connections is gathered for the (IP address, FQDN) pair under consideration.

If the number of connections from the source to the destinations in the set of network logs under consideration is greater than the strobe connection limit, the FQDN beacon is marked as a strobe. These hosts can be considered to have been in constant communication.

### Timestamp Beaconing Statistics
Inputs:
- `ParseResults.HostnameMap` created by `FSImporter`
    - Field: `Host`
        - Type: string
- MongoDB `hostname` collection:
    - Field: `host`
        - Type: string
    - Array Field: `dat`
        - Array Field: `ips`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
- MongoDB `uconn` collection:
    - Field: `src`
        - Type: string
    - Field: `src_network_uuid`
        - Type: UUID
    - Field: `dst`
        - Type: string
    - Field: `dst_network_uuid`
        - Type: UUID
    - Array Field: `dat`
        - Array Field: `ts`
            - Type: int64

Outputs:
- MongoDB `beaconFQDN` collection:
    - Array Field: `ts.intervals`
        - Type: int64
    - Array Field: `ts.interval_counts`
        - Type: int64
    - Field: `ts.range`
        - Type: int64
    - Field: `ts.mode`
        - Type: int64
    - Field: `ts.mode_count`
        - Type: int64
    - Field: `ts.dispersion`
        - Type: int64
    - Field: `ts.skew`
        - Type: float64

First, a set of (IP address, IP address) unique connections is gathered for the (IP address, FQDN) pair under consideration.

The `dat.ts` fields from the pair's `uconn` documents are unioned together in order to find all of the timestamps of the connections from the source to the destinations. 

After gathering all of the timestamps, the intervals between subsequent connections are derived by differencing the dataset. A frequency table is then constructed of the intervals and stored in the pair of fields: `ts.intervals` and `ts.interval_counts`. 

Given the dataset of connection intervals, the following statistics are derived:
- Range: Distance from the largest interval to the smallest interval
    - Field: `ts.range`
- Mode: Interval that appears the most often
    - Field: `ts.mode`
- Mode Count: How often the mode appears in the dataset
    - Field: `ts.mode_count`
- Dispersion: Median Absolute Deviation (MAD) around the median of intervals
    - Find the median of the intervals
        - Median: The value in the dataset such that half of the dataset is smaller than it
    - Find the distance from each interval to the median
    - Find the median of the distances
    - [MAD median on Wikipedia](https://en.wikipedia.org/wiki/Median_absolute_deviation)
    - Field: `ts.dispersion`
- Skew: Bowley Skew of the intervals
    - Find the median of the intervals (AKA the second quartile)
    - Find the first quartile of the intervals
        - Find the value in the dataset such that a quarter of the dataset is smaller than it
    - Find the third quartile of the intervals
        - Find the value in the dataset such that three quarters of the dataset is smaller than it
    - Skew = `(Q1 - 2 * Q2 + Q3) / (Q3 - Q1)`
        - `Q1` is the first quartile, `Q2` is the second, `Q3` is the third
    - Takes on values between -1 and 1, with 0 meaning the distribution of the dataset is symmetric
    - [Wikipedia gives a short explanation for Bowley Skew](https://en.wikipedia.org/wiki/Skewness#Quantile-based_measures)
    - Field: `ts.skew`

### Data Size Beaconing Statistics
Inputs:
- `ParseResults.HostnameMap` created by `FSImporter`
    - Field: `Host`
        - Type: string
- MongoDB `hostname` collection:
    - Field: `host`
        - Type: string
    - Array Field: `dat`
        - Array Field: `ips`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
- MongoDB `uconn` collection:
    - Field: `src`
        - Type: string
    - Field: `src_network_uuid`
        - Type: UUID
    - Field: `dst`
        - Type: string
    - Field: `dst_network_uuid`
        - Type: UUID
    - Array Field: `dat`
        - Array Field: `ds`
            - Type: int64

Outputs:
- MongoDB `beaconFQDN` collection:
    - Array Field: `ds.sizes`
        - Type: int64
    - Array Field: `ds.counts`
        - Type: int64
    - Field: `ds.range`
        - Type: int64
    - Field: `ds.mode`
        - Type: int64
    - Field: `ds.mode_count`
        - Type: int64
    - Field: `ds.dispersion`
        - Type: int64
    - Field: `ds.skew`
        - Type: float64

First, a set of (IP address, IP address) unique connections is gathered for the (IP address, FQDN) pair under consideration.

The `dat.bytes` fields from the pair's `uconn` documents are concatenated together in order to find all of the originating bytes of the connections from the source to the destinations. 

A frequency table is then constructed of the data sizes and stored in the pair of fields: `ds.sizes` and `ds.counts`. 

Given the dataset of data sizes, the following statistics are derived as above for timestamp intervals:
- Range: Distance from the largest data size to the smallest data size
    - Field: `ds.range`
- Mode: Data size that appears the most often
    - Field: `ds.mode`
- Mode Count: How often the mode appears in the dataset
    - Field: `ds.mode_count`
- Dispersion: Median Absolute Deviation (MAD) around the median of data sizes
    - Field: `ds.dispersion`
- Skew: Bowley Skew of the data sizes
    - Field: `ds.skew`

### Beacon Scoring
Inputs:
- `ParseResults.HostnameMap` created by `FSImporter`
    - Field: `Host`
        - Type: string
- MongoDB `hostname` collection:
    - Field: `host`
        - Type: string
    - Array Field: `dat`
        - Array Field: `ips`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
- MongoDB `uconn` collection:
    - Field: `src`
        - Type: string
    - Field: `src_network_uuid`
        - Type: UUID
    - Field: `dst`
        - Type: string
    - Field: `dst_network_uuid`
        - Type: UUID
    - Array Field: `dat`
        - Array Field: `ds`
            - Type: int64

Outputs:
- MongoDB `beaconFQDN` collection:
    - Field: `ts.conns_score`
        - Type: float64
    - Field: `ts.score`
        - Type: float64
    - Field: `ds.score`
        - Type: float64
    - Field: `score`
        - Type: float64

`ts.conns_score` records the ratio of the number of connections to the number of 10 second periods in the whole dataset. The score is capped at 1.

`ts.score` is calculated as `(1/3) * [(1 - |TS Bowley Skew|) + max(1 - (TS MADM)/30, 0) + (TS Conn. Count Score)]`.

`ds.score` is calculated as `(1/3) * [(1 - |DS Bowley Skew|) + max(1 - (DS MADM)/32, 0) + max(1 - (DS Mode) / 65535, 0)]`

### Highest Scoring FQDN Beacon Summary
Inputs: 
- `ParseResults.HostMap` created by `FSImporter`
    - Field: `IsLocal`
        - Type: bool
    - Field: `Host`
        - Type: data.UniqueIP
- MongoDB `beaconFQDN` collection:
    - Field: `src`
        - Type: string
    - Field: `src_network_uuid`
        - Type: UUID
    - Field: `src_network_name`
        - Type: string
    - Field: `fqdn`
        - Type: string
    - Field: `cid`
        - Type: int
    - Field: `score`
        - Type: float64

Outputs:
- Array Field: `dat`
    - Object Field: `mbfqdn`
        - Field: `ip`
            - Type: string
        - Field: `network_uuid`
            - Type: UUID
        - Field: `network_name`
            - Type: string
    - Field: `max_beacon_fqdn_score`
        - Type: int
    - Field: `cid`
        - Type: int

After building the `beaconFQDN` collection, RITA finds the FQDN with the highest beacon score for each of the internal hosts.

The `host` record's `dat.mbfqdn` field stores the FQDN of the FQDN beacon with the highest `score` in which internal host took part. The `dat.max_beacon_fqdn_score` field stores the associated `score` value. This analysis only considers beacons updated in the current chunk.

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the highest scoring FQDN beacon for an internal host, the maximum of the these subdocuments must be taken.