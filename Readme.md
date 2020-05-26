# RITA (Real Intelligence Threat Analytics)

[![RITA Logo](rita-logo.png)](https://www.activecountermeasures.com/free-tools/rita/)

Brought to you by [Active Countermeasures](https://www.activecountermeasures.com/).

---

RITA is an open source framework for network traffic analysis.

The framework ingests [Bro/Zeek Logs](https://www.zeek.org/) in TSV format, and currently supports the following major features:
 - **Beaconing Detection**: Search for signs of beaconing behavior in and out of your network
 - **DNS Tunneling Detection** Search for signs of DNS based covert channels
 - **Blacklist Checking**: Query blacklists to search for suspicious domains and hosts

## Install

Please see our recommended [System Requirements](docs/System%20Requirements.md) document if you wish to use RITA in a production environment.

### Automated Install

RITA provides an install script that works on Ubuntu 18.04 LTS, Ubuntu 16.04 LTS, Security Onion, and CentOS 7.

Download the latest `install.sh` file [here](https://github.com/activecm/rita/releases/latest) and make it executable: `chmod +x ./install.sh`

Then choose one of the following install methods:

* `sudo ./install.sh` will install RITA as well as supported versions of Bro/Zeek and MongoDB. This is suitable if you want to get started as quickly as possible or you don't already have Bro/Zeek or MongoDB.

* `sudo ./install.sh --disable-bro --disable-mongo` will install RITA only, without Bro/Zeek or MongoDB. You may also use these flags individually.
  * If you choose not to install Bro/Zeek you will need to [provide your own logs](#obtaining-data-generating-brozeek-logs).
  * If you choose not to install MongoDB you will need to configure RITA to [use your existing MongoDB server](docs/Mongo%20Configuration.md).

### Docker Install

See [here](docs/Docker%20Usage.md).

### Manual Installation

To install each component of RITA by manually see [here](docs/Manual%20Installation.md).

### Upgrading RITA

See [this guide](docs/Upgrading.md) for upgrade instructions.

### Getting Started

#### Configuration File

RITA's config file is located at `/etc/rita/config.yaml` though you can specify a custom path on individual commands with the `-c` command line flag.

:exclamation: **IMPORTANT** :exclamation:
* The `Filtering: InternalSubnets` section *must* be configured or you will not see any results in certain modules (e.g. beacons, long connections). If your network uses the standard RFC1918 internal IP ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16) you don't need to do anything as the default `InternalSubnets` section already has these. Otherwise, adjust this section to match your environment. RITA's main purpose is to find the signs of a compromised internal system talking to an external system and will automatically exclude internal to internal connections and external to external connections from parts of the analysis.

You may also wish to change the defaults for the following option:
* `Filtering: AlwaysInclude` - Ranges listed here are exempt from the filtering applied by the `InternalSubnets` setting. The main use for this is to include internal DNS servers so that you can see the source of any DNS queries made.

Note that any value listed in the `Filtering` section should be in CIDR format. So a single IP of `192.168.1.1` would be written as `192.168.1.1/32`.

#### Obtaining Data (Generating Bro/Zeek Logs)

  * **Option 1**: Generate PCAPs outside of Bro/Zeek
    * Generate PCAP files with a packet sniffer ([tcpdump](http://www.tcpdump.org/), [wireshark](https://www.wireshark.org/), etc.)
    * (Optional) Merge multiple PCAP files into one PCAP file
      * `mergecap -w outFile.pcap inFile1.pcap inFile2.pcap`
    * Generate Bro/Zeek logs from the PCAP files
      * ```bro -r pcap_to_log.pcap local "Log::default_rotation_interval = 1 day"```

  * **Option 2**: Install Bro/Zeek and let it monitor an interface directly [[instructions](https://docs.zeek.org/en/master/quickstart/index.html)]
      * You may wish to [compile Bro/Zeek from source](https://docs.zeek.org/en/master/install/install.html) for performance reasons. [This script](https://github.com/activecm/bro-install) can help automate the process.
      * The automated installer for RITA installs pre-compiled Bro/Zeek binaries by default
        * Provide the `--disable-bro` flag when running the installer if you intend to compile Bro/Zeek from source

#### Importing and Analyzing Data With RITA

After installing RITA, setting up the `InternalSubnets` section of the config file, and collecting some Bro/Zeek logs, you are ready to begin hunting.

RITA can process TSV, JSON, and [JSON streaming](https://github.com/corelight/json-streaming-logs) Bro/Zeek log file formats. These logs can be either plaintext or gzip compressed.

##### One-Off Datasets

This is the simplest usage and is great for analyzing a collection of Bro/Zeek logs in a single directory. If you expect to have more logs to add to the same analysis later see the next section on Rolling Datasets.

```
rita import path/to/your/bro_logs dataset_name`
```

Every log file in the supplied directory will be imported into a dataset with the given name. However, files in nested directories will not be processed.

##### Rolling Datasets

Rolling datasets allow you to progressively analyze log data over a period of time as it comes in.

```
rita import --rolling /path/to/your/bro_logs dataset_name
```

You can make this call repeatedly as new logs are added to the same directory (e.g. every hour).

One common scenario is to have a rolling database that imports new logs every hour and always has the last 24 hours worth of logs in it. Typically, Bro/Zeek logs will be placed in `/opt/bro/logs/<date>` which means that the directory will change every day. To accommodate this, you can use the following command in a cron job or other task scheduler that runs once per hour.

```
rita import --rolling /opt/bro/logs/$(date --date='-1 hour' +\%Y-\%m-\%d)/ dataset_name
```

RITA cycles data into and out of rolling databases in "chunks". You can think of each chunk as one hour, and the default being 24 chunks in a dataset. This gives the ability to always have the most recent 24 hours' worth of data available. But chunks are generic enough to accommodate non-default Bro logging configurations or data retention times as well. See the [Rolling Datasets](docs/Rolling%20Datasets.md) documentation for advanced options.

#### Examining Data With RITA

  * Use the **show-X** commands
      * `show-databases`: Print the datasets currently stored
      * `show-beacons`: Print hosts which show signs of C2 software
      * `show-bl-hostnames`: Print blacklisted hostnames which received connections
      * `show-bl-source-ips`: Print blacklisted IPs which initiated connections
      * `show-bl-dest-ips`: Print blacklisted IPs which received connections
      * `show-exploded-dns`:  Print dns analysis. Exposes covert dns channels
      * `show-long-connections`: Print long connections and relevant information
      * `show-strobes`: Print connections which occurred with excessive frequency
      * `show-useragents`: Print user agent information
  * By default RITA displays data in CSV format
      * `-H` displays the data in a human readable format
      * Piping the human readable results through `less -S` prevents word wrapping
          * Ex: `rita show-beacons dataset_name -H | less -S`
  * Create a html report with `html-report`

### Getting help

Please create an issue on GitHub if you have any questions or concerns.

### Contributing to RITA

To contribute to RITA visit our [Contributing Guide](Contributing.md)

### License

GNU GPL V3
&copy; Active Countermeasures &trade;
