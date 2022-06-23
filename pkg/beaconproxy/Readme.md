## HTTP Proxy Beacon Package

*Documented on June 1, 2022*

---
This package analyzes connections between internal hosts and fully qualified domain names (FQDNs) which are made using an HTTP proxy for signs of regular, programmatic communication.

This package records the following:
- The IP address, FQDN pair that communicated
- The IP address of the last proxy which serviced the connections
- Summary statistics of the connections between the pair
- Timestamp beaconing statistics
- Beacon scoring results

## Package Outputs

### Source Unique IP, Destination FQDN Pair
Inputs:
- `ParseResults.ProxyUniqueConnMap` created by `FSImporter`
    - Field: `Hosts`
        - Type: data.UniqueSrcFQDNPair

Outputs:
- MongoDB `beaconProxy` collection:
    - Field: `src`
        - Type: string
    - Field: `src_network_uuid`
        - Type: UUID
    - Field: `src_network_name`
        - Type: string
    - Field: `fqdn`
        - Type: string

The `src` field records the string representation of the IP address of the source of a proxied connection as seen in the network logs. Similarly, the field `fqdn` specifies the fully qualified domain name of the destination of a proxied connection as seen in the network logs. The `src_network_uuid` and `src_network` name fields have been introduced to disambiguate hosts using the same private IP address on separate networks. 

These fields are used to select an individual entry in the `uconnProxy` collection. All of the other outputs described here use the `src`, `src_network_uuid`, and `fqdn` fields as selectors when updating `beaconProxy` collection entries in MongoDB.

### Chunk ID
Inputs: 
- `Config.S.Rolling.CurrentChunk`
    - Type: int

Outputs:
- MongoDB `beaconProxy` collection:
    - Field: `cid`
        - Type: int

The `cid` field records the chunk ID of the import session in which this unique connection document was last updated. This field is used to support rolling imports.

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

### Unique Connection Summary Statistics
Inputs: 
- `ParseResults.ProxyUniqueConnMap` created by `FSImporter`
    - Field: `Hosts`
        - Type: data.UniqueSrcFQDNPair
- MongoDB `uconnProxy` collection:
    - Array Field: `dat`
        - Field: `count`
            - Type: int
            
Outputs:
- MongoDB `beaconProxy` collection:
    - Field: `connection_count`
        - Type: int

The `dat.count` fields from the pair's corresponding `uconnProxy` document are summed together in order to find the total amount of connections from the source IP address to the destination IP. The result is stored in the `connection_count` field of the pair's `beaconProxy` document.

### Timestamp Beaconing Statistics
Inputs:
- `ParseResults.ProxyUniqueConnMap` created by `FSImporter`
    - Field: `Hosts`
        - Type: data.UniqueSrcFQDNPair
- MongoDB `uconnProxy` collection:
    - Array Field: `dat`
        - Array Field: `ts`
            - Type: int64

Outputs:
- MongoDB `beaconProxy` collection:
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

The `dat.ts` fields from the pair's `uconnProxy` document are unioned together in order to find all of the timestamps of the connections from the source to the destination. 

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

### Beacon Scoring
Inputs:
- `ParseResults.ProxyUniqueConnMap` created by `FSImporter`
    - Field: `Hosts`
        - Type: data.UniqueSrcFQDNPair
- MongoDB `uconnProxy` collection:
    - Array Field: `dat`
        - Array Field: `ts`
            - Type: float64
        - Field: `count`
            - Type: int

Outputs:
- MongoDB `beaconProxy` collection:
    - Field: `ts.conns_score`
        - Type: float64
    - Field: `ts.score`
        - Type: float64
    - Field: `score`
        - Type: float64

`ts.conns_score` records the ratio of the number of connections to the number of 10 second periods in the whole dataset. The score is capped at 1.

`ts.score` is calculated as `(1/3) * [(1 - |TS Bowley Skew|) + max(1 - (TS MADM)/30, 0) + (TS Conn. Count Score)]`.

### Highest Scoring FQDN Beacon Summary
Inputs: 
- `ParseResults.HostMap` created by `FSImporter`
    - Field: `IsLocal`
        - Type: bool
    - Field: `Host`
        - Type: data.UniqueIP
- MongoDB `beaconProxy` collection:
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
        - Object Field: `mbproxy`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
            - Field: `network_name`
                - Type: string
        - Field: `max_beacon_proxy_score`
            - Type: int
        - Field: `cid`
            - Type: int

After building the `beaconProxy` collection, RITA finds the FQDN with the highest beacon score for each of the internal hosts.

The `host` record's `dat.mbproxy` field stores the FQDN of the proxy beacon with the highest `score` in which internal host took part. The `dat.max_beacon_proxy_score` field stores the associated `score` value. This analysis only considers beacons updated in the current chunk.

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the highest scoring proxy beacon for an internal host, the maximum of the these subdocuments must be taken.