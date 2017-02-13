#!/usr/bin/env bash
#
# RITA is brought to you by Offensive CounterMeasures.
# offensivecountermeasures.com

set -o errexit
set -o pipefail

_NAME=$(basename "${0}")
_INSDIR="/usr/local"

__help() {
	__title
	cat <<HEREDOC

Welcome to the RITA installer.

Usage:
	${_NAME} [<arguments>]
	${_NAME} -h | --help

Options:
	-h --help 		Show this help message.
	-i --install-dir	Directory to install to.

HEREDOC
}

__prep() {
	cat <<HEREDOC
So here's what this script will need to do to prepare for RITA:

1) Download and install GNU Netcat, Bro, Golang, and the latest version of MongoDB.

The MongoDB, netcat and golang versions we'd like aren't a part of the regular Ubuntu apt packages, but this script will add the key to the latest MongoDB repo to your package manager and install/auto config it and everything else.

2) Set up a Golang development enviornment in order to 'go get' and 'build' RITA.

This requires us to create directory "go" in your home folder and add a new PATH and GOPATH entry to your .bashrc

HEREDOC
}

__title() {
	cat <<HEREDOC

 _ \ _ _| __ __|  \
   /   |     |   _ \
_|_\ ___|   _| _/  _\

Brought to you by the Offensive CounterMeasures

HEREDOC
}

__install() {
	__title

	_RITADIR="$_INSDIR/rita"
	if [ -e $_RITADIR ]
	then
		printf "[+] $_RITADIR already exists.\n"
		read -p "     [-] Would you like to erase it and re-install? [y/n] " -n 1 -r
		if [[ $REPLY =~ ^[Yy]$ ]]
		then
			printf "\n[+] Removing $_RITADIR\n"
			rm -rf $_RITADIR
		else
			exit -1
		fi
	fi

  sudo apt update

  sudo apt install -y bro
  sudo apt install -y broctl
  sudo apt install -y build-essential


  if [ ! -f "/usr/local/go/"]
  then
      # golang most recent update
      wget https://storage.googleapis.com/golang/go1.7.1.linux-amd64.tar.gz
      sudo tar -zxvf  go1.7.1.linux-amd64.tar.gz -C /usr/local/
      sudo rm go1.7.1.linux-amd64.tar.gz
  else
        echo -e "\e[31m[-] WARNING: Go has been detected in /usr/bin/go,\e[37m if you
  installed with apt, RITA has only been tested with golang 1.7 which is currently not the
  version in the Ubuntu apt repositories, make sure your golang is up to date
  with 'go version'. Otherwise you can remove with 'sudo apt remove golang' and let this script
  install the correct version for you!"
      sleep 10s
  fi

  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

  echo -e "
[+] Done! Now just need to configure Go dev environment...

  \e[0m"

  sleep 3s

  if [[ -z "${GOPATH}" ]];
  then
    mkdir -p $HOME/go/{src,pkg,bin}
    echo 'export GOPATH=$HOME/go' >> $HOME/.bashrc
    export GOPATH=$HOME/go
    echo 'export PATH=$PATH:$GOPATH/bin' >> $HOME/.bashrc
    export PATH=$PATH:$GOPATH/bin
  else
    echo -e "[+] GOPATH seems to be set, we'll skip this part then for now
    "
  fi

  echo -e "[+] Now we need to get package key and MongoDB package...
"

  sleep 3s

  sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv 0C49F3730359A14518585931BC711F9BA15703C6

  echo "deb [ arch=amd64,arm64 ] http://repo.mongodb.org/apt/ubuntu xenial/mongodb-org/3.4 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-3.4.list

  sudo apt update
  sudo apt install -y mongodb-org

  sudo mkdir -p /data/db
  sudo chown -R $USER /data

  printf "[+] Running 'go get github.com/ocmdev/rita'\n\n"
  if [-e "/usr/local/go/bin"]
  then
    /usr/local/go/bin/go get github.com/ocmdev/rita
    cd $GOPATH/src/github.com/ocmdev/rita

    printf "[+] Done! Now we just have to build and install RITA.\n"
    /usr/local/go/bin/go build
    /usr/local/go/bin/go install
  else
    go get github.com/ocmdev/rita
    cd $GOPATH/src/github.com/ocmdev/rita
    go build
    go install
  fi

	printf "[+] Transferring files\n"
	mkdir $_RITADIR

	cp -r etc $_RITADIR/etc
	cp LICENSE $_RITADIR/LICENSE

	# Install the base configuration file
	if [ -w /etc/ ]
	then
		if [ ! -e /etc/rita ]
		then
			printf "[+] Installing global config to /etc/rita/config.yaml"
			mkdir /etc/rita
			cp etc/rita.yaml /etc/rita/config.yaml
		fi
	else
		if [ ! -e $HOME/.rita ]
		then
			printf "[+] Could not write to /etc installing local config in $HOME/.rita\n"
			cp etc/rita.yaml $HOME/.rita
		fi
	fi

  printf "[+] Installing gnu-netcat to /usr/local/rita\n"
  # gnu-netcat
  wget https://sourceforge.net/projects/netcat/files/netcat/0.7.1/netcat-0.7.1.tar.gz
  tar -zxf netcat-0.7.1.tar.gz
  rm netcat-0.7.1.tar.gz
  cd netcat-0.7.1
  ./configure --prefix=/usr/local/rita
  sudo make
  sudo make install
  cd ..
  rm -rf netcat-0.7.1

  # Give ownership of ~/go to the user
  sudo chown -R $USER /home/$USER/go

  echo "[+] Make sure you also configure Bro and run with 'sudo broctl deploy' and make sure MongoDB is running with the command 'mongo' or 'sudo mongo'.
"



  echo -e "
[+] If you need to stop Mongo at any time, run 'sudo service mongod stop'
[+] In order to finish the installation, reload bash config with 'source ~/.bashrc'.
[+] Also make sure to start the mongoDB service with 'sudo service mongod start before running RITA.
[+] You can access the mongo shell with 'sudo mongo'
"

	printf "Thank you for installing RITA!\n"
	printf "OCMDev Group projects IRC #ocmdev on OFTC\n"
	printf "Happy hunting\n"

}

# start point for installer
__entry() {

	# Check for help or other install dir
	if [[ "${1:-}" =~ ^-h|--help$ ]]
	then
		__help
	elif [[ "${1:-}" =~ ^-i|--install-dir ]]
	then
		_INSDIR=$( echo "${@}" | cut -d' ' -f2 )
	fi

	# Check to see if the user has permission to install to this directory
	if [ -w $_INSDIR ]
	then
		__install
	else
		printf "You do NOT have permission to write to $_INSDIR\n\n"
		__help
	fi
}

__entry "${@:-}"
