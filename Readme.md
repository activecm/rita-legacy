# RITA (Real Intelligence Threat Analytics)

Brought to you by [Active Countermeasures](https://www.activecountermeasures.com/).

---
### What is Here

RITA is an open source framework for network traffic analysis.

The framework ingests [Bro/Zeek Logs](https://www.zeek.org/), and currently supports the following major features:
 - **Beaconing Detection**: Search for signs of beaconing behavior in and out of your network
 - **DNS Tunneling Detection** Search for signs of DNS based covert channels
 - **Blacklist Checking**: Query blacklists to search for suspicious domains and hosts

### Automatic Installation
**The automatic  installer is officially supported on Ubuntu 16.04 LTS, Security Onion, and CentOS 7**

* Download the latest `install.sh` file from the [release page](https://github.com/activecm/rita/releases/latest)
* Make the installer executable: `chmod +x ./install.sh`
* Run the installer: `sudo ./install.sh`

### Manual Installation
To install each component of RITA by hand, [check out the instructions in the docs](docs/Manual%20Installation.md).

### Getting Started
#### System Requirements
* Operating System - The preferred platform is 64-bit Ubuntu 16.04 LTS. The system should be patched and up to date using apt-get.
* Processor (when also using Bro) - Two cores plus an additional core for every 100 Mb of traffic being captured. (three cores minimum). This should be dedicated hardware, as resource congestion with other VMs can cause packets to be dropped or missed.
* Memory - 16GB minimum. 64GB if monitoring 100Mb or more of network traffic. 128GB if monitoring 1Gb or more of network traffic.
* Storage - 300GB minimum. 1TB or more is recommended to reduce log maintenance.
* Network - In order to capture traffic with Bro, you will need at least 2 network interface cards (NICs). One will be for management of the system and the other will be the dedicated capture port. Intel NICs perform well and are recommended.

#### Upgrading RITA
See [this guide](docs/Upgrading.md) for upgrade instructions.

#### Configuration File
RITA's config file is located at `/etc/rita/config.yaml` though you can specify a custom path on individual commands with the `-c` command line flag.

:exclamation: **IMPORTANT** :exclamation:
* The `Filtering: InternalSubnets` section *must* be configured or you will not see any results in certain modules (e.g. beacons, long connections). If your network uses the standard RFC1918 internal IP ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16) you just need uncomment the default `InternalSubnets` section already in the config file. Otherwise, adjust this section to match your environment. RITA's main purpose is to find the signs of a compromised internal system talking to an external system and will automatically exclude internal to internal connections and external to external connections from parts of the analysis.

You may also wish to change the defaults for the following option:
* `Filtering: AlwaysInclude` - Ranges listed here are exempt from the filtering applied by the `IntenalSubnets` setting. The main use for this is to include internal DNS servers so that you can see the source of any DNS queries made.

Note that any value listed in the `Filtering` section should be in CIDR format. So a single IP of `192.168.1.1` would be written as `192.168.1.1/32`.

#### Obtaining Data (Generating Bro Logs):
  * **Option 1**: Generate PCAPs outside of Bro
    * Generate PCAP files with a packet sniffer ([tcpdump](http://www.tcpdump.org/), [wireshark](https://www.wireshark.org/), etc.)
    * (Optional) Merge multiple PCAP files into one PCAP file
      * `mergecap -w outFile.pcap inFile1.pcap inFile2.pcap`
    * Generate bro logs from the PCAP files
      * ```bro -r pcap_to_log.pcap local "Log::default_rotation_interval = 1 day"```

  * **Option 2**: Install Bro and let it monitor an interface directly [[instructions](https://www.bro.org/sphinx/quickstart/)]
      * You may wish to [compile Bro from source](https://www.bro.org/sphinx/install/install.html) for performance reasons. [This script](https://github.com/activecm/bro-install) can help automate the process.
      * The automated installer for RITA installs pre-compiled Bro binaries

#### Importing Data Into RITA
  * After installing, `rita` should be in your `PATH` and the config file should be set up ready to go. Once your Bro install has collected some logs (Bro will normally rotate logs on the hour) you can run `rita import`. Alternatively, you can manually import existing logs using one of the following options:
    * **Option 1**: Import directly from the terminal (one time import)
      * `rita import path/to/your/bro_logs/ database_name`
    * **Option 2**: Set up the Bro configuration in `/etc/rita/config.yaml` for repeated imports
      * Set `ImportDirectory` to the `path/to/your/bro_logs`. The default is `/opt/bro/logs`
      * Set `DBName` to an identifier common to your set of logs
  * Filtering and whitelisting of connection logs happens at import time, and those optional settings can be found in the `/etc/rita/config.yaml` configuration file.

#### Analyzing Data With RITA
  * **Option 1**: Analyze one dataset
    * `rita analyze dataset_name`
    * Ex: `rita analyze MyCompany_A`
  * **Option 2**: Analyze all imported datasets
    * `rita analyze`

#### Examining Data With RITA
  * Use the **show-X** commands
  * `-H` displays human readable data
  * `rita show-beacons dataset_name -H`
  * `rita show-blacklisted dataset_name -H`
  * Use less to view data `rita show-beacons dataset_name -H | less -S`

### Getting help
Please create an issue on GitHub if you have any questions or concerns.

### Contributing to RITA
To contribute to RITA visit our [Contributing Guide](Contributing.md)

### License
GNU GPL V3
&copy; Active Countermeasures &trade;
