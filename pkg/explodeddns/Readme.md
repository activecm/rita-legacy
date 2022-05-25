## Exploded DNS Package

*Documented on May 24, 2022*

---
This package counts the number of subdomains associated with each domain in the network logs under consideration. We call this analysis exploded DNS analysis since the first step of the analysis entails splitting each domain name into its constituent parts. 

For example, if we see the following fully qualified domain names in the network logs:

- a.b.c.com
- b.b.c.com
- c.c.c.com

The exploded DNS analysis produce the following results:

| Domain    | Subdomain Count |
|-----------|-----------------|
| com       | 3               |
| c.com     | 3               |
| b.c.com   | 2               |
| c.c.com   | 1               |
| a.b.c.com | 1               |
| b.b.c.com | 1               |
| c.c.c.com | 1               |

RITA refers to these domains which include the top level domain as "superdomains".

This package records the following:
- The superdomains under analysis
- How many subdomains were seen in the network logs for the given superdomain
- How often the superdomain or any of its subdomains were queried for in the network logs under consideration

## Package Outputs

### Superdomain Name

Inputs:
- `map[string]int` created by `FSImporter`
    - Key: Queried FQDN as seen in the network DNS logs
    - Value: How many times the FQDN was seen in the network DNS logs

Outputs:
- MongoDB `explodedDns` collection:
    - Field: `domain`
        - Type: string

Each FQDN seen as part of a DNS query in the network logs under consideration produces several records in the `explodedDns` collection. For example, processing `a.b.c.com` will result in creating four separate documents in the `explodedDns` collection. The `domain` field will be set to `a.b.c.com`, `b.c.com`, `c.com`, and `com` in the resulting documents.

### Chunk ID
Inputs: 
- `Config.S.Rolling.CurrentChunk`
    - Type: int

Outputs:
- MongoDB `explodedDns` collection:
    - Field: `cid`
        - Type: int

The `cid` field records the chunk ID of the import session in which this unique connection document was last updated. This field is used to support rolling imports.

### Subdomain Counts

Inputs:
- `map[string]int` created by `FSImporter`
    - Key: Queried FQDN as seen in the network DNS logs
    - Value: How many times the FQDN was seen in the network DNS logs

Outputs:
- MongoDB `explodedDns` collection:
    - Field: `subdomain_count`
        - Type: int

Each queried FQDN generates a set of superdomains or suffixes split on the `.` character. For example, `a.b.c.com` generates `a.b.c.com`, `b.c.com`, `c.com`, and `com`. For each superdomain, RITA increments the `subdomain_count` field in the corresponding `explodedDns` collection document.

In order to ensure this increment is idempotent when RITA chunks up network logs due to resource constraints, the `explodeddns` analysis must be run before the `hostname` analysis.

### Visited Counts

Inputs:
- `map[string]int` created by `FSImporter`
    - Key: Queried FQDN as seen in the network DNS logs
    - Value: How many times the FQDN was seen in the network DNS logs

Outputs:
- MongoDB `explodedDns` collection:
    - Array Field: `dat`
        - Field: `visited`
            - Type: int
        - Field: `cid`
            - Type: int

The `dat.visited` field tracks how many times a given superdomain was queried in the network logs under consideration. For example, if we see the following:
- 3 queries for `a.b.com`
- 2 queries for `z.b.com`
- 1 query for `c.com`

The following visited counts will be stored in MongoDB.

| Domain    | Visited Count |
|-----------|---------------|
| com       | 6             |
| b.com     | 5             |
| c.com     | 1             |
| a.b.com   | 3             |
| z.b.com   | 2             |