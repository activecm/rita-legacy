> :exclamation: **Important Notice** :exclamation:
> This repository has been archived and is no longer maintained. The project has undergone a complete rewrite and significant improvements. The new version of this project can be found [here](https://github.com/activecm/rita).

# RITA (Real Intelligence Threat Analytics) (Legacy)

[![RITA Logo](rita-logo.png)](https://www.activecountermeasures.com/free-tools/rita/)

If you get value out of RITA and would like to go a step further with hunting automation, futuristic visualizations, and data encrichment take a look at [AC-Hunter](https://www.activecountermeasures.com/).

Sponsored by [Active Countermeasures](https://activecountermeasures.com/).

---

RITA is an open source framework for network traffic analysis.

The framework ingests [Zeek Logs](https://www.zeek.org/) in TSV format, and currently supports the following major features:
 - **Beaconing Detection**: Search for signs of beaconing behavior in and out of your network
 - **DNS Tunneling Detection** Search for signs of DNS based covert channels
 - **Blacklist Checking**: Query blacklists to search for suspicious domains and hosts

## Install

Please see our recommended [System Requirements](docs/System%20Requirements.md) document if you wish to use RITA in a production environment.

### Automated Install

RITA provides an install script that works on Ubuntu 20.04 LTS, Debian 11, Security Onion, and CentOS 7.

Download the latest `install.sh` file [here](https://github.com/activecm/rita-legacy/releases/latest) and make it executable: `chmod +x ./install.sh`

Then choose one of the following install methods:

* `sudo ./install.sh` will install RITA as well as supported versions of Zeek and MongoDB. This is suitable if you want to get started as quickly as possible or you don't already have Zeek or MongoDB.

* `sudo ./install.sh --disable-zeek --disable-mongo` will install RITA only, without Zeek or MongoDB. You may also use these flags individually.
  * If you choose not to install Zeek you will need to [provide your own logs](#obtaining-data-generating-zeek-logs).
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

* The `Filtering: InternalSubnets` section *must* be configured or you will not see any results in certain modules (e.g. beacons, long connections). If your network uses the standard RFC1918 internal IP ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16) you don't need to do anything as the default `InternalSubnets` section already has these. Otherwise, adjust this section to match your environment. RITA's main purpose is to find the signs of a compromised internal system talking to an external system and will automatically exclude internal to internal connections and external to external connections from parts of the analysis.

You may also wish to change the defaults for the following option:
* `Filtering: AlwaysInclude` - Ranges listed here are exempt from the filtering applied by the `InternalSubnets` setting. The main use for this is to include internal DNS servers so that you can see the source of any DNS queries made.

Note that any value listed in the `Filtering` section should be in CIDR format. So a single IP of `192.168.1.1` would be written as `192.168.1.1/32`.

#### Obtaining Data (Generating Zeek Logs)

  * **Option 1**: Generate PCAPs outside of Zeek
    * Generate PCAP files with a packet sniffer ([tcpdump](http://www.tcpdump.org/), [wireshark](https://www.wireshark.org/), etc.)
    * (Optional) Merge multiple PCAP files into one PCAP file
      * `mergecap -w outFile.pcap inFile1.pcap inFile2.pcap`
    * Generate Zeek logs from the PCAP files
      * ```zeek -r pcap_to_log.pcap local "Log::default_rotation_interval = 1 day"```

  * **Option 2**: Install Zeek and let it monitor an interface directly [[instructions](https://docs.zeek.org/en/master/quickstart/index.html)]
      * You may wish to [compile Zeek from source](https://docs.zeek.org/en/master/install/install.html) for performance reasons. [This script](https://github.com/activecm/bro-install) can help automate the process.
      * The automated installer for RITA installs pre-compiled Zeek binaries by default
        * Provide the `--disable-zeek` flag when running the installer if you intend to compile Zeek from source
      * To take advantage of the feature for monitoring long-running, open connections (default is 1 hour or more), you will need to install our [zeek-open-connections plugin](https://github.com/activecm/zeek-open-connections/). We recommend installing the package with Zeek's package manager _zkg_. Newer versions of Zeek (4.0.0 or greater) will come bundled with _zkg_. If you do not have _zkg_ installed, you can [manually install](https://docs.zeek.org/projects/package-manager/en/stable/quickstart.html) it. Once you have _zkg_ installed, run the following commands to install the package
        * ```zkg refresh```
        * ```zkg install zeek/activecm/zeek-open-connections```

        Next, edit your site/local.zeek file so that it contains the following line
        * ```@load packages ```

        Finally, run the following
        * ```zeekctl deploy```

#### Importing and Analyzing Data With RITA

After installing RITA, setting up the `InternalSubnets` section of the config file, and collecting some Zeek logs, you are ready to begin hunting.

RITA can process TSV, JSON, and [JSON streaming](https://github.com/corelight/json-streaming-logs) Zeek log file formats. These logs can be either plaintext or gzip compressed.

##### One-Off Datasets

This is the simplest usage and is great for analyzing a collection of Zeek logs in a single directory. If you expect to have more logs to add to the same analysis later see the next section on Rolling Datasets.

```
rita import path/to/your/zeek_logs dataset_name
```

Every log file in the supplied directory will be imported into a dataset with the given name. However, files in nested directories will not be processed.

> :grey_exclamation: **Note:** Rita is designed to analyze 24hr blocks of logs. Rita versions newer than 4.5.1 will analyze only the most recent 24 hours of data supplied.

##### Rolling Datasets

Rolling datasets allow you to progressively analyze log data over a period of time as it comes in.

```
rita import --rolling /path/to/your/zeek_logs dataset_name
```

You can make this call repeatedly as new logs are added to the same directory (e.g. every hour).

One common scenario is to have a rolling database that imports new logs every hour and always has the last 24 hours worth of logs in it. Typically, Zeek logs will be placed in `/opt/zeek/logs/<date>` which means that the directory will change every day. To accommodate this, you can use the following command in a cron job or other task scheduler that runs once per hour.

```
rita import --rolling /opt/zeek/logs/$(date --date='-1 hour' +\%Y-\%m-\%d)/ dataset_name
```

RITA cycles data into and out of rolling databases in "chunks". You can think of each chunk as one hour, and the default being 24 chunks in a dataset. This gives the ability to always have the most recent 24 hours' worth of data available. But chunks are generic enough to accommodate non-default Zeek logging configurations or data retention times as well. See the [Rolling Datasets](docs/Rolling%20Datasets.md) documentation for advanced options.


> :grey_exclamation: **Note:** `dataset_name` is simply a name of your choosing. We recommend a descriptive name such as the hostname or location of where the data was captured. Stick with letters, numbers, and underscores. Periods and other special characters are not allowed.


#### Examining Data With RITA

  * Use the **show-X** commands
      * `show-databases`: Print the datasets currently stored
      * `show-beacons`: Print hosts which show signs of C2 software
      * `show-bl-hostnames`: Print blacklisted hostnames which received connections
      * `show-bl-source-ips`: Print blacklisted IPs which initiated connections
      * `show-bl-dest-ips`: Print blacklisted IPs which received connections
      * `show-dns-fqdn-ips`: Print IPs associated with a specified FQDN
      * `show-exploded-dns`:  Print dns analysis. Exposes covert dns channels
      * `show-long-connections`: Print long connections and relevant information
      * `show-strobes`: Print connections which occurred with excessive frequency
      * `show-useragents`: Print user agent information
  * By default, RITA displays data in CSV format
      * `-d [DELIM]` delimits the data by `[DELIM]` instead of a comma
          * Strings can be provided instead of single characters if desired, e.g. `rita show-beacons -d "---" dataset_name`
      * `-H` displays the data in a human readable format
          * This takes precedence over the `-d` option
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
