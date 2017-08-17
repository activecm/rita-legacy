#!/usr/bin/env bash
#
# RITA is brought to you by Offensive CounterMeasures.
# offensivecountermeasures.com

_NAME=$(basename "${0}")
_FAILED="\e[91mFAILED\e[0m"
_SUCCESS="\e[92mSUCCESS\e[0m"

#Error handling
#Kill 0 to kill subshells as well
trap "printf '\n[!] Installation $_FAILED!\n'; kill 0" ERR INT
set -o errexit
set -o errtrace
set -o pipefail

# Make sure to source the latest .bashrc
# Hack the PS1 variable to get around ubuntu .bashrc
OLD_PS1=$PS1
PS1=" "
# Hack the interactive flag to get around other .bashrc's
set -i

source $HOME/.bashrc

# Clean up our hacks
set +i
PS1=$OLD_PS1
unset OLD_PS1


__help() {
	__title

	cat <<HEREDOC
Usage:
	${_NAME} [<arguments>]

Options:
	-h --help			Show this help message.
	-u --uninstall			Remove RITA.

HEREDOC
}

__prep() {
	cat <<HEREDOC
This script will:

1) Download and install Bro IDS, Go, and MongoDB.

2) Set up a Go development enviornment in order to 'go get'
and 'build' RITA. This requires us to create a directory "go"
in your home folder and add new PATH and GOPATH entries
to your .bashrc.

3) Create a configuration directory for RITA in your home folder called .rita

HEREDOC

	sleep 5s
}

__title() {
	echo \
"
 _ \ _ _| __ __|  \\
   /   |     |   _ \\
_|_\ ___|   _| _/  _\\

Brought to you by Offensive CounterMeasures
"

}

__load() {
  local pid=$!
  local loadingText=$1

  while kill -0 $pid 2>/dev/null; do
    echo -ne "$loadingText.\r"
    sleep 0.5
    echo -ne "$loadingText..\r"
    sleep 0.5
    echo -ne "$loadingText...\r"
    sleep 0.5
    echo -ne "\r\033[K"
    echo -ne "$loadingText\r"
    sleep 0.5
  done
	wait $pid
  echo -e "$loadingText... $_SUCCESS"
}

__checkPermissions() {
	[ `id -u` -eq 0 ]
}

__uninstall() {
	printf "\t[!] Removing $GOPATH/bin/rita \n"
	rm -rf $GOPATH/bin/rita
	printf "\t[!] Removing $GOPATH/src/github.com/ocmdev \n"
	rm -rf $GOPATH/src/github.com/ocmdev
	printf "\t[!] Removing $HOME/.rita \n"
	rm -rf $HOME/.rita
}

__install() {

	# Check if RITA is already installed, if so ask if this is a re-install
	if [ ! -z $(command -v rita) ] ||
	[ -f $GOPATH/bin/rita ]
	then
		printf "[+] RITA is already installed.\n"
		read -p "[-] Would you like to erase it and re-install? [y/n] " -r
		if [[ $REPLY =~ ^[Yy]$ ]]
		then
			__uninstall
		else
			exit -1
		fi
	fi

	__prep

	# Install installation dependencies
	apt-get update > /dev/null & __load "[+] Updating apt"

	apt-get install -y git wget make lsb-release > /dev/null & \
	__load "[+] Installing git, wget, make, and lsb-release"

  # Install Bro IDS
	printf "[+] Checking if Bro IDS is installed... "

	if [ $(dpkg-query -W -f='${Status}' bro 2>/dev/null | grep -c "ok installed") -eq 0 ] &&
	[ $(dpkg-query -W -f='${Status}' securityonion-bro 2>/dev/null | grep -c "ok installed") -eq 0 ]
	then
		printf "\n"
		apt-get install -y bro broctl bro-aux > /dev/null & \
		__load "\t[+] Installing Bro IDS"
	else
		printf "$_SUCCESS\n"
	fi

  # Install Go
	printf "[+] Checking if Go is installed...\n"

	# Check if go is not available in the path
	if [ -z $(command -v go) ]
	then
		# Check if go is available in the standard location
		if [ ! -e "/usr/local/go" ]
		then
			( # golang most recent update
				wget -q https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz
				tar -zxf go1.8.3.linux-amd64.tar.gz -C /usr/local/
				rm go1.8.3.linux-amd64.tar.gz
				echo 'export PATH=$PATH:/usr/local/go/bin' >> $HOME/.bashrc
			) &	__load "\t[+] Installing Go"
		fi

		# Add go to the path
		export PATH="$PATH:/usr/local/go/bin"
	else
		echo -e "\e[93m\t[!] WARNING: Go has been detected on this system.\e[0m
\tIf you installed Go with apt, make sure your Go installation is up
\tto date with 'go version'. RITA has only been tested with golang
\t1.7 and 1.8 which are currently not the versions in the Ubuntu
\tapt repositories. You may remove the old version with
\t'sudo apt remove golang' and let this script install the correct
\tversion for you!
"
		sleep 10s
	fi

	# Check if the GOPATH isn't set
	if [ -z ${GOPATH+x} ]
	then
		( # Set up the GOPATH
			mkdir -p $HOME/go/{src,pkg,bin}
			echo 'export GOPATH=$HOME/go' >> $HOME/.bashrc
			echo 'export PATH=$PATH:$GOPATH/bin' >> $HOME/.bashrc
		) & __load "[+] Configuring Go dev environment"
		export GOPATH=$HOME/go
		export PATH=$PATH:$GOPATH/bin
	fi

	# Install MongoDB
	apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 \
	--recv 0C49F3730359A14518585931BC711F9BA15703C6 > /dev/null 2>&1 & \
	__load "[+] Obtaining the package key for MongoDB"

	echo "deb [ arch=$(dpkg --print-architecture) ] http://repo.mongodb.org/apt/ubuntu $(lsb_release -cs)/mongodb-org/3.4 multiverse" > /etc/apt/sources.list.d/mongodb-org-3.4.list

	apt-get update > /dev/null & __load "[+] Updating apt"
	apt-get install -y mongodb-org > /dev/null & __load "[+] Installing MongoDB"

	( # Build RITA
		go get github.com/ocmdev/rita
		cd $GOPATH/src/github.com/ocmdev/rita
		make install > /dev/null
	)	& __load "[+] Installing RITA"


	( # Install the base configuration files
		mkdir $HOME/.rita
		mkdir $HOME/.rita/logs
		cd $GOPATH/src/github.com/ocmdev/rita
		cp ./LICENSE $HOME/.rita/LICENSE
		cp ./etc/rita.yaml $HOME/.rita/config.yaml
		cp ./etc/tables.yaml $HOME/.rita/tables.yaml
	) & __load "[+] Installing config files to $HOME/.rita"


	# If the user is using sudo, give ownership to the sudo user
	if [ -z ${SUDO_USER+x} ]
	then
		chown -R $SUDO_USER:$SUDO_USER $HOME/go
		chown -R $SUDO_USER:$SUDO_USER $HOME/.rita
	fi

	echo -e "
In order to finish the installation, reload your bash config
with 'source ~/.bashrc'. Make sure to configure Bro and run
'sudo broctl deploy'. Also, make sure to start the MongoDB
service with 'sudo service mongod start'. You can access
the MongoDB shell with 'mongo'. If, at any time, you need
to stop MongoDB, run 'sudo service mongod stop'."

	__title
	printf "Thank you for installing RITA! "
	printf "Happy hunting\n"
}

# start point for installer
__entry() {

	# Check for help
	if [[ "${1:-}" =~ ^-h|--help$ ]]
	then
		__help
		exit 0
	fi

	# Check to see if the user has permission to install RITA
	if __checkPermissions
	then
		# Check if we are uninstalling
		if [[ "${1:-}" =~ ^-u|--uninstall ]]
		then
			__uninstall
		else
			__install
		fi
	else
		printf "You do NOT have permission install RITA\n\n"
	fi
}

__entry "${@:-}"
