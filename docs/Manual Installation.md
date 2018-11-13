
### Installation

This guide walks through installing several components.

* [Bro/Zeek](https://www.bro.org)
* [MongoDB](https://www.mongodb.com)
* [RITA](https://github.com/activecm/rita/)

#### Bro/Zeek

Installing Bro is recommended. RITA needs Bro logs as input so if you already have Bro or its logs you can skip installing Bro.

1. Follow the directions at [https://www.bro.org/sphinx/install/install.html](https://www.bro.org/sphinx/install/install.html).
1. Use Bro's quick start guide to configure [https://www.bro.org/sphinx-git/quickstart/index.html](https://www.bro.org/sphinx-git/quickstart/index.html).

#### MongoDB

RITA requires Mongo for storing and processing data. The current recommended version is 3.6, but anything >= 3.2.0 and < 3.7.0 should work.

1. Follow the MongoDB installation guide at https://docs.mongodb.com/manual/installation/
    * Alternatively, this is a direct link to the [download page](https://www.mongodb.com/download-center?jmp=nav#community)
1. Ensure MongoDB is running before running RITA.  

#### RITA

You have a few options for installing RITA.
1. The main install script. You can disable Bro and Mongo from being installed with the `--disable-bro` and `--disable-mongo` flags.
1. A prebuilt binary is available for download on [RITA's release page](https://github.com/activecm/rita/releases). In this case you will need to download the config file from the same release and create some directories manually, as described below in the "Configuring the system" section.
1. [Use RITA with docker](Docker%20Usage.md)
1. Compile RITA manually from source. See below.

##### Installing Golang

In order to compile RITA manually you will need to install both [Golang](https://golang.org) and [Dep](https://github.com/golang/dep).

1. Install Golang using the instructions at [https://golang.org/doc/install](https://golang.org/doc/install)
1. After the install you need to create a local Go development environment for your user. This is typically done in `$HOME/go` which is what the directions here will use.
    1. ```mkdir -p $HOME/go/{src,pkg,bin}```
1. Now you must add the `GOPATH` to your .bashrc file. You will also want to add your bin folder to the path for this user.
    1. ```echo 'export GOPATH="$HOME/go"' >> $HOME/.bashrc```
    1. ```echo 'export PATH="$PATH:$GOPATH/bin"' >> $HOME/.bashrc```
    1. ```source $HOME/.bashrc```
1. Install the depenency manager dep using [these instructions](https://golang.github.io/dep/docs/installation.html)

##### Building RITA

At this point you can build RITA from source code.

1. ```go get github.com/activecm/rita```
1. ```cd $GOPATH/src/github.com/activecm/rita```
1. ```make``` (Note that you will need to have `make` installed. You can use your system's package manager to install it.)

This will yield a `rita` binary in the current directory. You can use `make install` to install the binary to `$GOPATH/bin/rita` or manually copy/link it to `/usr/local/bin/rita` or another location you desire.

##### Configuring the system

RITA requires a few directories to be created for it to function correctly.

1. ```sudo mkdir /etc/rita && sudo chmod 755 /etc/rita```
1. ```sudo mkdir -p /var/lib/rita/logs && sudo chmod -R 755 /var/lib/rita```
1. ```sudo touch /var/lib/rita/safebrowsing && sudo chmod 666 /var/lib/rita/safebrowsing```

Copy the config file from your local RITA source code.
* ```sudo cp $GOPATH/src/github.com/activecm/rita/etc/rita.yaml /etc/rita/config.yaml && sudo chmod 666 /etc/rita/config.yaml```

At this point, you can modify the config file as needed and test using the ```rita test-config``` command. There will be empty quotes or 0's assigned to empty fields. [RITA's readme](../Readme.md) has more information on changing the configuration.
