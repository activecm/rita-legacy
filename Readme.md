#RITA

Brought to you by Offensive CounterMeasures

###What's here

RITA has all of the logic used to analyze Bro data. With an input of Bro data a
MongoDB database will be created, which can be analyzed for review of that data.
All of the mathematics, lookups, and storage of Offensive CounterMeasures AI
Hunter is available in this package. The only thing not here is the graphical
front end which Offensive CounterMeasures has created to help visualize this
data.

### Automatic Installation
Rita is currently supported on Ubuntu 16.04 LTS.

Clone the package:
```bash
git clone https://github.com/ocmdev/rita.git
```

Change into the source directory:
```bash
cd rita
```
Run the installer:

**Note:** Make sure you have permission to write to the install directory.
By default, Rita will install to /usr/local/rita.
However, you can change the install location with the *-i* flag.
```bash
./install.sh
```

***or***

```bash
./install.sh -i /path/to/install/directory
```

### Manual Installation
To install each component of Rita by hand, [check out the instructions in the wiki](https://github.com/ocmdev/rita/wiki/Installation).

### Configuration File
The default location for the config file is */etc/rita/config.yaml*. However, if the installer did not have permission to write to that directory, the config file will be found at *~/.rita*.


### API Keys
Rita relies on the the [Google Safe Browsing API](https://developers.google.com/safe-browsing/) to check network log data for connections to known threats. An API key is required to use this service. Obtaining a key is free, and only requires a Google account.

To obtain an API key:
  * Go to the [cloud platform console](https://console.cloud.google.com/).
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
**Note: WORKING ON THIS SECTION**
 * Will describe collecting logs, importing, and analyzing.
 * Also planning on writing an entry in the wiki on setting up a tap/span port and collecting bro logs.

###Getting help
Head over to OFTC and join #ocmdev for any questions you may have. Please
remember that this is an open source project, the developers working in here
have full time jobs and are not your personal tech support. So please be civil
with us.

###License
GNU GPL V3
&copy; Offensive CounterMeasures &trade;

###Contributing

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

1. Helping add features:
  	* If you'd like to become involved in the development effort please hop on our
OFTC channel at #ocmdev and try and chat with booodead about what's currently
being worked on.
  	* If you have a feature request or idea, also please hop on OFTC #ocmdev and
chat with booodead about your idea. There's a chance we're already working on it and
would be happy to share that work with you.

#####Submitting work:
Please send pull requests and such as small as possible. As this is a product that
we use internally, as well as a backend for a piece of commercially supported
software. Every line of code that goes in must be inspected and approved. So if it
is taking a while to get back to you on your work, or we reject code, don't be
offended, we're just paranoid and desire to get this project to a very stable and
useable place.
