
### Installation

1. What you'll need:
    * Bro [https://www.bro.org](https://www.bro.org)
    * MongoDB [https://www.mongodb.com](https://www.mongodb.com)
    * Golang [https://www.golang.org](https://www.golang.org)
1. Install Bro [Optional]:
    1. Follow the directions at [https://www.bro.org/sphinx/install/install.html](https://www.bro.org/sphinx/install/install.html)
    1. Test that bro is working by firing up bro and ensuring that it's spitting out logs. If you're having some trouble with bro configuration or use, here are some helpful links:
        * Bro quick start [https://www.bro.org/sphinx-git/quickstart/index.html](https://www.bro.org/sphinx-git/quickstart/index.html)
        * broctl [https://www.bro.org/sphinx/components/broctl/README.html](https://www.bro.org/sphinx/components/broctl/README.html)
1. Install MongoDB (You will need a version between 3.2.0 and 3.7.0 which is not included by default in the Ubuntu 16.04 package manager.)
    * Follow the MongoDB installation guide at https://docs.mongodb.com/manual/installation/
    * Download a version >= 3.2.0, but < 3.7.0 at https://www.mongodb.com/download-center?jmp=nav#community
    * Ensure MongoDB is running before continuing  
1. Install GoLang using the instructions at [https://golang.org/doc/install](https://golang.org/doc/install)
    1. After the install we need to set a local GOPATH for our user. So lets set up a directory in our HomeDir
        * ```mkdir -p $HOME/go/{src,pkg,bin}```
    1. Now we must add the GoPath to our .bashrc file
        * ```echo 'export GOPATH="$HOME/go"' >> $HOME/.bashrc```
    1. We will also want to add our bin folder to the path for this user.
        * ```echo 'export PATH="$PATH:$GOPATH/bin"' >> $HOME/.bashrc```
    1. Load your new configurations with source.
        * ```source $HOME/.bashrc```
1. Getting RITA and building it
  	1. First we want to use the go to grab sources and deps for rita.
    	* ```go get github.com/activecm/rita```
  	1. Now lets change to the rita directory.
    	* ```cd $GOPATH/src/github.com/activecm/rita```
  	1. Finally we'll build and install the rita binary.
  		* ```make install```
		* This will install to `$GOPATH/bin/rita` not `/usr/local/bin/rita`
1. Configuring the system
    1. Create a configuration directory at `/etc/rita`
        * ```sudo mkdir /etc/rita```
    1. Allow users to read the configuration directory
        * ```sudo chmod 755 /etc/rita```
    1. Create a runtime directory for rita at `/var/lib/rita`
        * ```sudo mkdir -p /var/lib/rita/logs```
    1. Allow users to write to the runtime directory
        * ```sudo chmod 755 /var/lib/rita```
        * ```sudo chmod 777 /var/lib/rita/logs```
    1. Create the safebrowsing database file
        * ```sudo touch /var/lib/rita/safebrowsing```
    1. Allow users to write to the safebrowsing file
        * ```sudo chmod 666 /var/lib/rita/safebrowsing```
    1. Install the config file
        * ```sudo cp etc/rita.yaml /etc/rita/config.yaml```
    1. Allow users to write to the RITA config file
        * ```sudo chmod 666 /etc/rita/config.yaml```
    1. You can test a configuration file with ```rita test-config -c PATH/TO/FILE```
        * There will be empty quotes or 0's assigned to empty fields
    1. Follow the documentation in the Readme.md for configuring RITA
