# RITA (Real Intelligence Threat Analytics)

Brought to you by Offensive CounterMeasures.

---
### What is Here

RITA is an open source framework for network traffic analysis.

The framework ingests [Bro Logs](https://www.bro.org/), and currently supports the following analysis features:
 - **Beaconing**: Search for signs of beaconing behavior in and out of your network
 - **DNS Tunneling** Search for signs of DNS based covert channels
 - **Blacklisted**: Query blacklists to search for suspicious domains and hosts
 - **URL Length Analysis**: Search for lengthy URLs indicative of malware
 - **Scanning**: Search for signs of port scans in your network

Additional functionality is being developed and will be included soon.

### Automatic Installation
**The automatic  installer is officially supported on Ubuntu 14.04, 16.04 LTS, Security Onion, and CentOS 7**

* Clone the package:
`git clone https://github.com/ocmdev/rita.git`
* Change into the source directory: `cd rita`
* Run the installer: `sudo ./install.sh`
* Source your .bashrc (the installer added RITA to the PATH): `source ~/.bashrc`
* Start MongoDB: `sudo service mongod start`

### Docker Installation
RITA is available as a Docker image at ocmdev/rita, [check out the instructions in the wiki](https://github.com/ocmdev/rita/wiki/Docker-Installation).

### Manual Installation
To install each component of RITA by hand, [check out the instructions in the wiki](https://github.com/ocmdev/rita/wiki/Installation).

### Configuration File
RITA contains a yaml format configuration file.

You can specify the location for the configuration file with the **-c** command line flag. If not specified, RITA will look for the configuration in **/etc/rita/config.yaml**.


### API Keys
RITA relies on the the [Google Safe Browsing API](https://developers.google.com/safe-browsing/) to check network log data for connections to known threats. An API key is required to use this service. Obtaining a key is free, and only requires a Google account.

To obtain an API key:
  * Go to the Google [cloud platform console](https://console.cloud.google.com/).
  * From the projects list, select a project or create a new one.
  * If the API Manager page is not already open, open the left side menu and select **API Manager**.
  * On the left, choose **Credentials**.
  * Click **Create credentials** and then select **API key**.
  * Copy this API key to the **APIKey** field under **SafeBrowsing** in the configuration file.
  * On the left, choose **Library**.
  * Search for **Safe Browsing**.
  * Click on **Google Safe Browsing API**.
  * Near the top, click **Enable**.

### Getting Started
#### Obtaining Data (Generating Bro Logs):
  * **Option 1**: Generate PCAPs outside of Bro
    * Generate PCAP files with a packet sniffer ([tcpdump](http://www.tcpdump.org/), [wireshark](https://www.wireshark.org/), etc.)
    * (Optional) Merge multiple PCAP files into one PCAP file
      * `mergecap -w outFile.pcap inFile1.pcap inFile2.pcap`
    * Generate bro logs from the PCAP files
      * Set local_nets to your local networks
      * ```bro -r pcap_to_log.pcap local "Site::local_nets += { 192.168.0.0/24 }"  "Log::default_rotation_interval = 1 day"```

  * **Option 2**: Install Bro and let it monitor an interface directly [[instructions](https://www.bro.org/sphinx/quickstart/)]
      * You may wish to [compile Bro from source](https://www.bro.org/sphinx/install/install.html) for performance reasons. [This script](https://github.com/ocmdev/bro-install) can help automate the process.
      * The automated installer for RITA installs pre-compiled Bro binaries

#### Importing Data Into RITA
  * After installing, `rita` should be in your `PATH` and the config file should be set up ready to go. Once your Bro install has collected some logs (Bro will normally rotate logs on the hour) you can run `rita import`. Alternatively, you can manually import existing logs using one of the following options:
  * **Option 1**: Import directly from the terminal (one time import)
    * `rita import path/to/your/bro_logs/ database_name`
  * **Option 2**: Set up the Bro configuration in `/etc/rita/config.yaml` for repeated imports
    * Set `ImportDirectory` to the `path/to/your/bro_logs`. The default is `/opt/bro/logs`
    * Set `DBRoot` to an identifier common to your set of logs

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

### Getting help
Head over to [OFTC and join #ocmdev](https://webchat.oftc.net/?channels=ocmdev) for any questions you may have.

### Contributing to RITA
To contribute to RITA visit our [Contributing Guide](https://github.com/ocmdev/rita/blob/master/Contributing.md)

### License
GNU GPL V3
&copy; Offensive CounterMeasures &trade;
