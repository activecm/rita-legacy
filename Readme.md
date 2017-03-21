# RITA

Brought to you by Offensive CounterMeasures.

### What's here

RITA is an open source framework for network traffic analysis.

The framework ingests [Bro Logs](https://www.bro.org/), and currently supports the following analysis features:
 - **Beaconing**: Search for signs of beaconing behavior in and out of your network
 - **Blacklisted**: Query blacklists to search for suspicious domains and hosts in your network traffic
 - **Scanning**: Search for signs of port scans in your network

Additional functionality is being developed and will be included soon.

### Automatic Installation
**The automatic RITA installer is officially supported on Ubuntu 16.04 LTS**

Clone the package:
```bash
git clone https://github.com/ocmdev/rita.git
```

Change into the source directory:
```bash
cd rita
```
Run the installer:

**Note:**
By default, Rita will install to /usr/local/rita.
However, you can change the install location with the *-i* flag.
```bash
sudo ./install.sh
```

***or***

```bash
sudo ./install.sh -i /path/to/install/directory
```

### Manual Installation
To install each component of Rita by hand, [check out the instructions in the wiki](https://github.com/ocmdev/rita/wiki/Installation).

### Configuration File
RITA contains a yaml format configuration file.

You can specify the location for the configuration file with the **-c** command line flag. If not specified, RITA will first look for the configuration in **~/.rita/config.yaml** then **/etc/rita/config.yaml**.


### API Keys
Rita relies on the the [Google Safe Browsing API](https://developers.google.com/safe-browsing/) to check network log data for connections to known threats. An API key is required to use this service. Obtaining a key is free, and only requires a Google account.

To obtain an API key:
  * Go to the Google [cloud platform console](https://console.cloud.google.com/).
  * From the projects list, select a project or create a new one.
  * If the API Manager page isn't already open, open the left side menu and select **API Manager**.
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
    * Merge your PCAP files into one PCAP
      * `mergecap -w outFile.pcap inFile1.pcap inFile2.pcap`
    * Generate bro logs from the PCAP files
      * `bro -r pcap_to_log.pcap`
  * **Option 2**: Install Bro and let it monitor an interface directly [[instructions](https://www.bro.org/sphinx/quickstart/)]
      * You may wish to [compile Bro from source](https://www.bro.org/sphinx/install/install.html) for performance reasons

#### Importing Data Into RITA
  * **Option 1**: Import directly from the terminal (one time import)
    * `rita import -i path/to/yout/bro_logs/ -d dataset_name`
  * **Option 2**: Set up the Bro configuration in config.yaml for repeated imports
    * Set `LogPath` to the `path/to/your/bro_logs`
    * Set `DBPrefix` to an identifier common to your sets of logs
    * Set up the `DirectoryMap`
      * Logs found in folders which match the substring on the left are imported
      into  the dataset on the right
    * Example
      * Let's say you have two sets of logs to analyze
        * `/share/bro_logs/networkA`
        * `/share/bro_logs/networkB`
      * A correct Bro config would look like
      ```yaml
      Bro:
        LogPath: /share/bro_logs/
          DBPrefix: MyCompany_
          DirectoryMap:
            networkA: A
            networkB: B
      ```
      * This would import `/share/brologs/networkA` into `MyCompany_A` and
      `/share/brologs/networkB` into `MyCompany_B`


#### Analyzing Data With RITA


**Link to video tutorial will be added soon!**

### Getting help
Head over to OFTC and join #ocmdev for any questions you may have.

### License
GNU GPL V3
&copy; Offensive CounterMeasures &trade;

### Contributing

Want to help? We'd love that! Here are some ways to get involved ranging in
difficulty from easiest to hardest.

1. Run the software and tell us when it breaks. We're happy to recieve bug
reports. Just be sure to do the following:
  	* Give very specific descriptions of how to reproduce the bug
  	* Let us know if you're running RITA on weird hardware
  	* Tell us about the size of the test, and the physical resources available

1. Add godoc comments to the code. This software was developed for internal use
mostly on the fly and as needed. This means that the code was not built to the
typical standards of an open source project and we would like to get it there.

1. Fix style compliance issues. Just run golint and start fixing non-compliant
code.

1. Work on bug fixes. Grab from the issues list and submit fixes.

1. Help add features:
  	* If you would like to become involved in the development effort, please hop on our
OFTC channel at #ocmdev and chat about what's currently being worked on.

#####Submitting work:
Please send pull requests and such as small as possible. As this is a product that
we use internally, as well as a backend for a piece of commercially supported
software. Every line of code that goes in must be inspected and approved. So if it
is taking a while to get back to you on your work, or we reject code, don't be
offended, we're just paranoid and desire to get this project to a very stable and
useable place.
