
### Installation

This guide walks through installing several components.

* [Zeek](https://www.zeek.org)
* [MongoDB](https://www.mongodb.com)
* [RITA](https://github.com/activecm/rita/)

#### Zeek

Installing Zeek is recommended. RITA needs Zeek logs as input so if you already have Zeek or its logs you can skip installing Zeek.

1. Follow the directions at https://zeek.org/get-zeek/.
1. Use the [quick start guide](https://docs.zeek.org/en/current/quickstart/index.html) to configure.

#### MongoDB

RITA requires Mongo for storing and processing data. The current supported version is 4.2, but anything >= 4.0.0 may work.

1. Follow the MongoDB installation guide at https://docs.mongodb.com/v4.2/installation/
    * Alternatively, this is a direct link to the [download page](https://www.mongodb.com/try/download/community). Be sure to choose version 4.2
1. Ensure MongoDB is running before running RITA.

#### RITA

You have a few options for installing RITA.
1. The main install script. You can disable Zeek and Mongo from being installed with the `--disable-zeek` and `--disable-mongo` flags.
1. A prebuilt binary is available for download on [RITA's release page](https://github.com/activecm/rita/releases). In this case you will need to download the config file from the same release and create some directories manually, as described below in the "Configuring the system" section.
1. Compile RITA manually from source. See below.

##### Installing Golang

In order to compile RITA manually you will need to install [Golang](https://golang.org/doc/install) (v1.13 or greater).

##### Building RITA

At this point you can build RITA from source code.

1. ```git clone https://github.com/activecm/rita.git```
1. ```cd rita```
1. ```make``` (Note that you will need to have `make` installed. You can use your system's package manager to install it.)

This will yield a `rita` binary in the current directory. You can use `make install` to install the binary to `/usr/local/bin/rita` or `PREFIX=/ make install` to install to a different location (`/bin/rita` in this case).

##### Configuring the system

RITA requires a few directories to be created for it to function correctly.

1. ```sudo mkdir /etc/rita && sudo chmod 755 /etc/rita```
1. ```sudo mkdir -p /var/lib/rita/logs && sudo chmod -R 755 /var/lib/rita```

Copy the config file from your local RITA source code.
* ```sudo cp etc/rita.yaml /etc/rita/config.yaml && sudo chmod 666 /etc/rita/config.yaml```

At this point, you can modify the config file as needed and test using the ```rita test-config``` command. There will be empty quotes or 0's assigned to empty fields. [RITA's readme](../Readme.md#configuration-file) has more information on changing the configuration.
