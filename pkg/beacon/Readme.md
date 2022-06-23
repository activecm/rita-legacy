## Beacon Package

*Documented on May 24, 2022*

---
This package analyzes connections between internal and external hosts for signs of regular, programmatic communication. 

This package records the following:
- The pair of IP addresses that communicated
- Summary statistics of the connections between the pair
- Timestamp beaconing statistics
- Data size beaconing statistics
- Beacon scoring results

## Package Outputs

### Source and Destination Unique IP Address Pair
Inputs: 
- `ParseResults.UniqueConnMap` created by `FSImporter`
    - Field: `Hosts`
        - Type: data.UniqueIPPair

Outputs:
- MongoDB `beacon` collection:
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

These fields are used to select an individual entry in the `beacon` collection. All of the other outputs described here use the `src`, `src_network_uuid`, `dst`, and `dst_network_uuid` fields as selectors when updating `beacon` collection entries in MongoDB.

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
- `ParseResults.UniqueConnMap` created by `FSImporter`
    - Field: `Hosts`
        - Type: data.UniqueIPPair
- MongoDB `uconn` collection:
    - Array Field: `dat`
        - Array Field: `bytes`
            - Type: int
        - Field: `count`
            - Type: int
        - Field: `tbytes`
            - Type: int
            
Outputs:
- MongoDB `beacon` collection:
    - Field: `connection_count`
        - Type: int
    - Field: `avg_bytes`
        - Type: float64
    - Field: `total_bytes`
        - Type: int

The `dat.count` fields from the pair's corresponding `uconn` document are summed together in order to find the total amount of connections from the source IP address to the destination IP. The result is stored in the `connection_count` field of the pair's `beacon` document.

Similarly, the `dat.tbytes` fields from the `uconn` document are summed together to find the total amount of bytes sent between the two hosts. The result is stored in the `total_bytes` field in the pair's `beacon` document.

The `dat.bytes` arrays from the `uconn` document are concatenated and the average of the values stored in the `avg_bytes` field of the pair's `beacon` document. Note that this is the average of the originating bytes, as opposed to the two way bytes tracked by `total_bytes`.

### Timestamp Beaconing Statistics
Inputs:
- `ParseResults.UniqueConnMap` created by `FSImporter`
    - Field: `Hosts`
        - Type: data.UniqueIPPair
- MongoDB `uconn` collection:
    - Array Field: `dat`
        - Array Field: `ts`
            - Type: int64

Outputs:
- MongoDB `beacon` collection:
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

The `dat.ts` fields from the pair's `uconn` document are unioned together in order to find all of the timestamps of the connections from the source to the destination. 

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
- `ParseResults.UniqueConnMap` created by `FSImporter`
    - Field: `Hosts`
        - Type: data.UniqueIPPair
- MongoDB `uconn` collection:
    - Array Field: `dat`
        - Array Field: `bytes`
            - Type: int64

Outputs:
- MongoDB `beacon` collection:
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

The `dat.bytes` fields from the pair's `uconn` document are concatenated together in order to find all of the originating bytes of the connections from the source to the destination. 

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
- `ParseResults.UniqueConnMap` created by `FSImporter`
    - Field: `Hosts`
        - Type: data.UniqueIPPair
- MongoDB `uconn` collection:
    - Array Field: `dat`
        - Array Field: `ts`
            - Type: float64
        - Array Field: `bytes`
            - Type: int
        - Field: `count`
            - Type: int
        - Field: `tbytes`
            - Type: int

Outputs:
- MongoDB `beacon` collection:
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

### Highest Scoring Beacon Summary

Inputs: 
- `ParseResults.HostMap` created by `FSImporter`
    - Field: `IsLocal`
        - Type: bool
    - Field: `Host`
        - Type: data.UniqueIP
- MongoDB `beacon` collection:
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
    - Field: `cid`
        - Type: int
    - Field: `score`
        - Type: float64

Outputs:
- Array Field: `dat`
    - Object Field: `mbdst`
        - Field: `ip`
            - Type: string
        - Field: `network_uuid`
            - Type: UUID
        - Field: `network_name`
            - Type: string
    - Field: `max_beacon_score`
        - Type: int
    - Field: `cid`
        - Type: int

After building the `beacon` collection, RITA finds the external host with the highest beacon score for each of the internal hosts.

The `host` record's `dat.mbdst` field stores the external IP address of the unique connection with the highest `score` in which internal host took part. The `dat.max_beacon_score` field stores the associated `score` value. This analysis only considers beacons updated in the current chunk.

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

Multiple subdocuments may be produced by a single run `rita import` if the import session had to be broken into several sessions due to resource considerations. In order to return the highest scoring beacon for an internal host, the maximum of the these subdocuments must be taken.