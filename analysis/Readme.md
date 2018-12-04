# Rita Analysis

## Analysis Packages
Rita contains several analysis packages designed to extract useful intelligence from raw bro logs.

- Unique Connections
  - Provides a list of who talked to whom in the dataset
- Hosts
  - Provides a list of ip addresses in the dataset
- Urls
  - Provides a list of url + uri pairs in the dataset
- Hostnames
  - Provides a mapping from hostnames to ip addresses
- Exploded DNS
  - Provides a count of how many subdomains were associated with a given domain name
- Beacons
  - Provides a statistical view on connections, looking for regularity
- Blacklisted
  - Provides a list of ip addresses and domain names that were blacklisted by other authorities
- User Agent
  - Provides a list of the user agent strings in the dataset and how many times they were used
- Internal Cross Reference
  - Provides an aggregate view of the other modules which are related to internal hosts
- External Cross Reference
  - Provides an aggregate view of the other modules which are related to external hosts
