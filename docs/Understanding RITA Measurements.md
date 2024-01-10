## Beacons
Compromised systems often send homing signals, known as 'beacons,' back to command and control servers. Attackers use beacons to monitor access to compromised machines as well as to transmit instructions, data, and additional malicious code. Because beacon connections are more uniform than their human-generated counterparts, beacon analysis may be used to identify suspicious patterns caused by malicious activity on a system.

##### Total Score (Score)
Total score (shown as score) is the average of the **interval score** and the **data size score**.
###### Interval Score
The score for interval is calculated by averaging each of the interval measurements (skew, dispersion, and connection count score).
**Skew** and **Dispersion** are described below.
**Connection count score** is calculated by comparing the number of connections to the range of intervals between connections.
###### Data Size Score
The score for data size is calculated by averaging data size skew, dispersion, and a smallness score.
**Skew** and **Dispersion** are described below
**Smallness score** is a score based on how big the mode of data sizes is. A beacon will often be sending very small amounts of data frequently, so smaller data size mode indicates a higher likelihood of beacon activity.

##### Source IP
This is the IP of the system that initiated the connection.

##### Destination IP
This is the IP of the system that received the connection.

##### Connections
Connections counts the number of times the source IP connected to the destination IP.

##### Average Bytes (Avg Bytes)
Average Bytes is the total number of bytes sent in both response and request connections divided by the number of connections.

##### Interval (Intvl) vs Data Size (Size)
There are two main ways beacons are looked for. One of them is by comparing the time between connections, which is referred to as **interval** (shown as **intvl**). The other is looking at the size of data being passed in each connection known as **data size** (shown as **size**).
In some cases attackers will disguise a beacon through intervals that are random enough, they might not appear to be beacons. In this case, these beacons can still be recognized because they have a consistent data size being passed.
Each of the following measurements have columns for both interval and data size.

##### Range
Range is the difference between the largest interval or data size, and the smallest interval or data size.

##### Top
Top is the mode of the interval or data size values.

##### Top Count
Top Count is the number of times the mode of the interval or data size was seen.

##### Skew
Skew measures how distorted or asymmetric the data is. More symmetric data would result in a skew value closer to 100%, while more asymmetric data would result in a skew value closer to 0%. This is useful in the case of malware that does not try to disguise its beacons, but is especially useful in detecting malware that has applied jitter to its beaconing. This is because jitter is typically implemented by using a random number generator to add or subtract to a mean value (e.g. 30 seconds +/- 5 seconds for the interval). The random number generator will end up uniformly distributing the values, which results in the data being symmetric.

##### Dispersion
Dispersion describes how likely the interval is to stray from the mean. A dispersion value closer to 100% means that most values were clustered around the mean, and had little variation. Typical traffic will have large variation in both the intervals between connections and the amount of data passed during the connection. In the case of obvious beacons, time intervals, and/or data size tend to be clustered around one value.
This is only useful in the case of beacons that weren't designed to be hard to find. The more jitter there is, the less useful dispersion is.


## Blacklist

Hostnames or IP addresses that are determined to be disreputable could wind up on a blacklist. The most common of these include servers that send spam, host malware, and originate phishing attacks. If your Bro/Zeek system collects packets outside of your firewall, most connection attempts here will (hopefully) be filtered out by your firewall and can be ignored. If, however, you are collecting packets inside of your firewall filtering, finding a connection to a blacklisted host or IP on your network could be a potential indicator that the machine which originated the connection has been compromised.

When viewing blacklisted results there are several shared fields:
##### Connections
The total number of times the corresponding blacklisted hostname or IP was connected to.

##### Unique Connections
The number of different systems that have connected to the corresponding blacklisted host or IP.

##### Total bytes
The total amount of data exchanged between the corresponding blacklisted host or IP, and internal systems.

##### Sources/Destinations
A list of systems that show connections to the blacklisted host or IP.

#### Hostnames
##### Host
The hostname, domain name, or FQDN that appears on at least one of the blacklists used.

#### Source Or Destination IP
The BL Source IPs lists blacklisted IPs that initiated connections with other systems. The BL Dest. IPs lists blacklisted IPs that received connections from other systems.

##### IP
The IP that appears on at least one of the blacklists used


## Exploded DNS
DNS is a fairly noisy protocol, and as such tends to have minimal logging enabled. This combined with the fact that most environments allow DNS out of their environment means that DNS can be used as a covert communication channel and a way to exfiltrate data out of a network. In general, most domains shouldn't have a lot of subdomains (in most cases probably fewer than 100). If a domain is spotted with an unusually high number of subdomains, and that domain isn't familiar, there's a good chance that DNS is being used for exfiltration.

##### Domain
This is the domain seen on the network (ex. example.com or 1.example.com).

##### Unique Subdomains
The number of subdomains of the corresponding domain seen on the network (ex. if the network shows connections to example.com, 1.example.com, 2.example.com, and 3.2.example.com, this value will be 4 for example.com, 1 for 1.example.com and 3.2.example.com, and 2 for 2.example.com).

##### Times Looked Up
Total count of connections made to the corresponding domain and its subdomains on the network (ex. if there was 1 connection to 1.example.com, 2 connections to 2.example.com, and 3 connections to 3.2.example.com this value will be 6 for example.com, 1 for 1.example.com, 5 for 2.example.com, and 3 for 3.2.example.com).


## Long Connections
Some attackers will attempt to leave connections open for long periods of time in an effort to evade beacon analysis while still remaining in contact with a compromised system. Looking for connections that have been open for hours or longer can help find these channels.

##### Source IP
The IP of the system that initiated the connection.

##### Destination IP
The IP of the system that received the connection.

##### Port:Protocol:Service
The port number and the names of the protocol and service used in the connection. If a connection is suspisciously long it makes sense to check if the combination of these three things makes sense.

##### Duration
The length of the connection in seconds.


## Strobes
Similar to beacons, strobes are repeated connections between two IP addresses. Unlike beacons, strobes make no effort to hide their signaling. An obvious example of a strobe is a signal that triggers two or three times a second.

##### Source
The IP of the system that initiated the connections.

##### Destination
The IP of the system that received the connections.

##### Connection Count
The number of connections made.


## User Agents
User agent names are sent as part of an HTTP request and are used to identify the browser or tool making the request. Unique user agents can be interesting in that they may identify a non-standard tool or browser being used on the network which may indicate someone may be using a compromised or malicious tool or browser.

##### User Agent
The string used as the user agent name.

##### Times Used
The number of times the corresponding user agent was seen.
