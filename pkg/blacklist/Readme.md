## Threat Intelligence Package

*Documented on June 7, 2022*

---

This packages summarizes the connections made between internal hosts and external hosts which appear on threat intelligence lists.

## Package Outputs

### Peer Connection Summary
Inputs:
- MongoDB `host` collection:
    - Field: `ip`
        - Type: string
    - Field: `network_uuid`
        - Type: UUID
    - Field: `network_name`
        - Type: string
    - Field: `blacklisted`
        - Type: bool
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
        - Field: `count`
            - Type: int
        - Field: `tbytes`
            - Type: int
    - Field: `open_bytes`
        - Type: int
    - Field: `open_connection_count`
        - Type: int

Outputs:
- MongoDB `host` collection:
    - Array Field: `dat`
        - Field: `bl`
            - Field: `ip`
                - Type: string
            - Field: `network_uuid`
                - Type: UUID
            - Field: `network_name`
                - Type: string
        - Field: `bl_conn_count`
            - Type: int
        - Field: `bl_total_bytes`
            - Type: int
        - Field: `bl_in_count`
            - Type: int
        - Field: `bl_out_count`
            - Type: int
        - Field: `cid`
            - Type: int

After the `host` package and `uconn` packages run, the threat intelligence package creates summaries for the hosts which were the connection peers of other hosts which were marked as unsafe.

Unsafe hosts are first gathered using the `host` collection. Then, the connection peers of these hosts are then queried using the `uconn` collection. The connection counts and bytes of each of these `uconn` entries are derived from their respective `dat` subdocuments. Finally, the results are stored in new `dat` subdocuments in the peers' `host` entries. 

`bl` stores the unsafe host this host contacted. `bl_conn_count` tracks how many times the hosts connected. `bl_total_bytes` tracks how many bytes were sent back and forth between the hosts. `bl_in_count` and `bl_out_count` are either absent or set to 1 in each `dat` subdocument. 

The current chunk ID is recorded in this subdocument in order to track when the entry was created.

There should always be one `dat` subdocument per unsafe host this host contacted. Multiple subdocuments with the same `bl` field should not exist.