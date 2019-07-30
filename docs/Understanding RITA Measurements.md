## Beacons

Compromised systems often send homing signals, known as 'beacons,' back to command and control servers. Attackers use beacons to monitor access to compromised machines as well as to transmit instructions, data, and additional malicious code. Because beacon connections are more uniform than their human-generated counterparts, beacon analysis may be used to identify suspicious patterns caused by malicious activity on a system.

#### Score (Total, Interval, and Data Size)

**Total Score** is the average of interval score and data size score. The closer this score is to 100% the more likely that this is beaconing activity.
**Interval Score** is the average of interval skew, interval dispersion, and interval duration. The closer this score is to 100%, the more the interval data indicates beaconing activity.
**Data Size Score** is the average of data size skew, data size dispersion, and data size mode. The closer this score is to 100%, the more the data sizes being received indicates beaconing activity.

#### Skew (Interval and Data Size)

Skew measures how distorted or asymmetric the data is. A value closer to 100% means that the data is very symmetric. This measure is useful in the case of malware that does not try to hide itself, but more importantly will detect malware that tries to hide by adding jitter. This works because malware with jitter most likely uses a random number generator to add or subtract to a mean value (e.g. 30 seconds +/- 5 seconds). The random number generator will uniformly distribute the values which causes the data to be symmetric, and as such this particular measure is hard to beat.

#### Dispersion (Interval and Size)

Dispersion describes how likely it is that an interval or data size is to stray from the mean (e.g. standard deviation from the mean). A value closer to 100% means that most intervals or data sizes were clustered around the same value and had very little variation. This is useful in the case of beacons that don't make any effort to hide themselves by changing their beacon interval or data sizes. The more jitter added to a beacon, the less effective dispersion is.

#### Duration (Interval)

Duration measures the time period the beacon was active for on the assumption that malware would be beaconing the entire time. A value closer to 100% would a beacon that is active for the entire analyzed time period. This measurement is susceptible to false positives, for example, if there are only 2 connections with one at the beginning and one at the end of the analyzed time period, duration would measure close to 100%. It is also susceptible to false negatives in the case of a beacon only being active for a few hours, which some malware can be configured to do, in which case the duration would measure close to 0%.

#### Mode (Interval and Data Size)

In the case of **interval** this is the mode of the intervals between connections. This value is printed out for the show-beacons command, but it is not used in any calculations.
In the case of **data size** this is the mode of the size of data sent (not received), the idea being that many small data sizes are more likely to be beacons. The closer to 100% this is, the closer to 0 bytes the data size mode is.

## Blacklist

Hostnames or IP addresses that are determined to be disreputable could wind up on a blacklist. The most common of these include servers that send spam, host malware, and originate phishing attacks. If your Bro/Zeek system collects packets outside of your firewall, most connection attempts here will (hopefully) be filtered out by your firewall and can be ignored. If, however, you are collecting packets inside of your firewall filtering, finding a connection to a blacklisted host or IP on your network could be a potential indicator that the machine which originated the connection has been compromised.

## Exploded DNS

DNS is a fairly noisy protocol, and as such tends to have minimal logging enabled. This combined with the fact that most environments allowing DNS out of their environment means that DNS can be used as a covert communication channel and a way to exfiltrate data out of a network.

#### Domain

This is the domain seen on the network (ex. example.com).

#### Unique Subdomains

The number of subdomains of the corresponding domain (ex. if the network shows connections to example.com, 1.example.com, 2.example.com, and 3.example.com, this value will be 4).

#### Times Looked Up

Total count of connections made to the corresponding domain and its subdomains on the network (ex. if there was 1 connection to 1.example.com, 2 connections to 2.example.com, and 3 connections to 3.example.com this value will be 6).

## Long Connections

Some attackers will attempt to leave connections open for long periods of time in an effort to evade beacon analysis. So looking for connections that have been open for hours or longer can help find these channels.

## Strobes

Similar to beacons, strobes are repeated connections between two IP addresses. Unlike beacons, strobes make no effort to hide their signaling. An obvious example of a strobe is a signal that triggers two or three times a second.

## User Agents

User agent names are sent as part of an HTTP request and are used to identify the browser or tool making the request. Unique user agents can be interesting in that they may identify a non-standard tool or browser being used on the network which may indicate someone may be using a compromised or malicious tool or browser.
