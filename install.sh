#!/usr/bin/env bash
#
# RITA is brought to you by Active CounterMeasures.
# activecountermeasures.com

_NAME=$(basename "${0}")
_FAILED="\e[91mFAILED\e[0m"
_SUCCESS="\e[92mSUCCESS\e[0m"

#Error handling
#Kill 0 to kill subshells as well
__err() {
	printf "\n[!] Installation $_FAILED!\n"
	kill 0
}
trap __err ERR INT
set -o errexit
set -o errtrace
set -o pipefail

# Fix $HOME for users under standard sudo
if [ ! -z ${SUDO_USER+x} ]; then
	HOME="$( getent passwd $SUDO_USER | cut -d: -f6 )"
fi

# Make sure to source the latest .bashrc
# Hack the PS1 variable to get around ubuntu .bashrc
_OLD_PS1=$PS1
PS1=" "
# Hack the interactive flag to get around other .bashrc's
set -i
# Make sure weirdness doesn't happen with autocomplete/ etc
set -o posix

source $HOME/.bashrc

# Clean up our hacks
set +o posix
set +i
PS1=$_OLD_PS1
unset _OLD_PS1

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

__mod_config() {
	bro_log_path=$1
	mongodb_conn_str=$2
	rita_config="$_CONFIG_PATH/config.yaml"

	printf "[+] Getting config values\n"

	printf "\t[-] Please Enter a Meta-Database Name (default is \"MetaDatabase\" press enter to keep default): "
	read meta_db
	if [ "$meta_db" == "" ]; then
		meta_db='MetaDatabase'
	fi

	printf "\t[-] Please Enter your Google Safe Browsing API Key (default is \"\" press enter to keep default): "
	read api_key

	#Leave one example and print the remaining directory structure
	sed -i "s+^    ConnectionString: mongodb://localhost:27017*+""    ConnectionString: $mongodb_conn_str""+" $rita_config
	sed -i "s+^    LogPath: /opt/bro/logs/*+""    LogPath: $bro_log_path""+" $rita_config
	sed -i "s/^    MetaDB: MetaDatabase*/""    MetaDB: $meta_db""/" $rita_config
	sed -i 's/^        APIKey: ""*/''        APIKey: "'$api_key'"''/' $rita_config

}

__explain() {
	cat <<HEREDOC
This script will:

1) Download and install Bro IDS, Go, and MongoDB.

2) Set up a Go development environment in order to install
RITA. This requires us to create new directories
in $_INSTALL_PREFIX and add new PATH and GOPATH entries
to your .bashrc.

3) Create a configuration directory for RITA in $_CONFIG_PATH

The following information is needed
	* Bro Log Path
	* MongoDB Connection String
	* Metadatabase Name
	* Google API Key (optional)

HEREDOC

	sleep 5s
}

__title() {
	echo \
"
 _ \ _ _| __ __|  \\
   /   |     |   _ \\
_|_\ ___|   _| _/  _\\

Brought to you by Active CounterMeasures
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
	if [ "$_INSTALL_PREFIX" != "/opt/rita" ]; then
		# Too risky to delete files if we don't know where it was installed (e.g. could have installed to /)
		printf "\t[!] Automatic uninstall from a non-standard location is not supported \n"
	else
		printf "\t[!] Removing /opt/rita \n"
		rm -rf /opt/rita
	fi
	printf "\t[!] Removing $_CONFIG_PATH \n"
	rm -rf "$_CONFIG_PATH"
}

__setPkgMgr() {
	# _PKG_MGR = 1: APT: Ubuntu 14.04, 16.04 and Security Onion (Debian)
	# _PKG_MGR = 2: YUM: CentOS (Old RHEL Derivatives)
	# _PKG_MGR = 3: Unsupported
	_PKG_MGR=3
	_PKG_INSTALL=""
	if [ -x /usr/bin/apt-get ];	then
		_PKG_MGR=1
		_PKG_INSTALL="apt-get -qq install -y"
	elif [ -x /usr/bin/yum ];	then
		_PKG_MGR=2
		_PKG_INSTALL="yum -y -q install"
	fi
	if [ $_PKG_MGR -eq 3 ]; then
		echo "Unsupported package manager"
		__err
	fi
}

__setOS() {
	_OS="$(lsb_release -is)"
	if [ "$_OS" != "Ubuntu" -a "$_OS" != "CentOS" -a "$_OS" != "LinuxMint"]; then
		echo "Unsupported operating system"
		__err
	fi
}

__install_packages() {
	while [ ! -z "$1" ]; do
		local pkg="$1"
		# Translation layer
		# apt -> yum
		if [ $_PKG_MGR -eq 2 ]; then
			case "$pkg" in
				"lsb-release")
					pkg="redhat-lsb-core"
					;;
				realpath)
					pkg="coreutils"
					;;
			esac
		fi
		eval $_PKG_INSTALL $pkg >/dev/null 2>&1
		shift
	done
}

__freshen_packages() {
	if [ $_PKG_MGR -eq 1 ]; then   #apt
		apt-get -qq update > /dev/null 2>&1
	elif [ $_PKG_MGR -eq 2 ]; then #yum
		yum -q makecache > /dev/null 2>&1
	fi
}

__package_installed() {
	#Returns true if the package is installed, false otherwise
	if [ $_PKG_MGR -eq 1 ]; then # apt
		dpkg-query -W -f='${Status}' "$1" 2>/dev/null | grep -q "ok installed"
	elif [ $_PKG_MGR -eq 2 ]; then # yum and dnf
		rpm -q "$1" >/dev/null
	fi
}

__add_deb_repo() {
	if [ ! -s "/etc/apt/sources.list.d/$2.list" ]; then
		if [ ! -z "$3" ]; then
			curl -s -L "$3" | apt-key add - > /dev/null 2>&1
		fi
		echo "$1" > "/etc/apt/sources.list.d/$2.list"
		__freshen_packages
	fi
}

__add_rpm_repo() {
	yum-config-manager -q --add-repo=$1 > /dev/null 2>&1
}

__check_go_version() {
	case `go version | awk '{print $3}'` in
	go1|go1.2*|go1.3*|go1.4*|go1.5*|go1.6*|"")
		echo -e "\e[93m\t[!] WARNING: Go has been detected on this system.\e[0m
\tIf you installed Go with apt, make sure your Go installation is up
\tto date with 'go version'. RITA has only been tested with golang
\t1.7 and 1.8 which are currently not the versions in the Ubuntu
\tapt repositories. You may remove the old version with
\t'sudo apt remove golang' and let this script install the correct
\tversion for you!
"
		sleep 10s
		;;
	esac
}

__install_go() {
	# Check if go isn't available in the path
	printf "[+] Checking if Go is installed...\n"
	if [ ! $(command -v go) ];	then
		if [ ! -x "/usr/local/go/bin/go" ]; then
			(
				curl -s -o /tmp/golang.tar.gz https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz
				tar -zxf /tmp/golang.tar.gz -C /usr/local/
				rm /tmp/golang.tar.gz
			) & __load "\t[+] Installing Go"
		fi
		printf "\t[+] Adding Go to the PATH...\n"
		export PATH="$PATH:/usr/local/go/bin"
		echo 'export PATH=$PATH:/usr/local/go/bin' >> $HOME/.bashrc
	else
		printf "\t[+] Go is installed...\n"
	fi

	# Check if the GOPATH isn't set
	if [ -z ${GOPATH+x} ]; then
		( # Set up the GOPATH
			mkdir -p $_INSTALL_PREFIX/{src,pkg,bin}
			#echo "export GOPATH=$_INSTALL_PREFIX" >> $HOME/.bashrc
			echo "export PATH=\$PATH:$_INSTALL_PREFIX/bin" >> $HOME/.bashrc
		) & __load "\t[+] Configuring Go dev environment"
		export GOPATH=$_INSTALL_PREFIX
		export PATH=$PATH:$_INSTALL_PREFIX/bin
	fi
}

__install_bro() {
	(
		# security onion packages bro on their own
		if ! __package_installed bro && ! __package_installed securityonion-bro; then
			case "$_OS" in
				Ubuntu)
					__add_deb_repo "deb http://download.opensuse.org/repositories/network:/bro/xUbuntu_$(lsb_release -rs)/ /" \
						"Bro" \
						"http://download.opensuse.org/repositories/network:bro/xUbuntu_$(lsb_release -rs)/Release.key"
					;;
				CentOS)
					__add_rpm_repo http://download.opensuse.org/repositories/network:bro/CentOS_7/network:bro.repo
					;;
			esac
			__install_packages bro broctl
		fi
	) & __load "[+] Ensuring Bro IDS is installed"

	if [ ! $(command -v bro) ];	then
		printf "\t[+] Adding Bro to the PATH...\n"
		echo 'export PATH=$PATH:/opt/bro/bin' >> $HOME/.bashrc
		PATH=$PATH:/opt/bro/bin
	fi
	chmod 2755 /opt/bro/logs
}

__install_mongodb() {
	case "$_OS" in
		Ubuntu)
			__add_deb_repo "deb [ arch=$(dpkg --print-architecture) ] http://repo.mongodb.org/apt/ubuntu $(lsb_release -cs)/mongodb-org/3.4 multiverse" \
				"MongoDB" \
				"https://www.mongodb.org/static/pgp/server-3.4.asc"
			;;
		CentOS)
			if [ ! -s /etc/yum.repos.d/mongodb-org-3.4.repo ]; then
				echo -e '[mongodb-org-3.4]\nname=MongoDB Repository\nbaseurl=https://repo.mongodb.org/yum/redhat/$releasever/mongodb-org/3.4/x86_64/\ngpgcheck=1\nenabled=1\ngpgkey=https://www.mongodb.org/static/pgp/server-3.4.asc' > /etc/yum.repos.d/mongodb-org-3.4.repo
			fi
			;;
	esac
	__install_packages mongodb-org
}

__install() {

	# Check if RITA is already installed, if so ask if this is a re-install
	if [ ! -z $(command -v rita) ] \
		|| [ -d /opt/rita ] \
		|| [ -d $_CONFIG_PATH ]
	then
		printf "[+] RITA is already installed.\n"
		read -p "[-] Would you like to erase it and re-install? [y/n] " -r
		if [[ $REPLY =~ ^[Yy]$ ]]
		then
			__uninstall
			echo ""
		else
			exit 1
		fi
	fi

	# Explain the scripts actions
	__explain

	# Figure out which package manager to use
	__setPkgMgr

	# Update package sources
	__freshen_packages

	# Install "the basics"
	__install_packages git wget curl make coreutils realpath lsb-release & \
		__load "[+] Ensuring git, wget, curl, make, coreutils, and lsb-release are installed"

	# Determine the OS, needs lsb-release
	__setOS

	bro_log_path="/opt/bro/logs/"
	if [[ "${_INSTALL_BRO}" = "true" ]]
	then
		if ! __package_installed bro && ! __package_installed securityonion-bro;
		then
			printf "[+] Bro is not installed\n"
			read -p "[-] Would you like to install Bro? [y/n] " -r
			if [[ $REPLY =~ ^[Yy]$ ]]; then
				__install_bro
				echo ""
			else
				printf "[-] Please ensure you install bro and modify rita.yaml before running RITA\n"
			fi
		else
			printf "[+] Bro is Installed\n"
			printf "[-] Please enter log path (default is \"/opt/bro/logs\" press enter to keep default): "
			read bro_log_path
			if [ "$bro_log_path" == "" ]; then
				bro_log_path='/opt/bro/logs'
			fi
		fi
	fi

  __install_go
	__check_go_version

	mongodb_conn_str="mongodb://localhost:27017"
	if [[ "${_INSTALL_MONGO}" = "true" ]]
	then
		if ! __package_installed mongodb-org; then
			printf "[+] MongoDB is not installed\n"
			read -p "[-] Would you like to install MongoDB [y/n] " -r
			if [[ $REPLY =~ ^[Yy]$ ]]; then
				__install_mongodb & __load "[+] Installing MongoDB"
			else
				printf "[-] Please ensure you install MongoDB and modify rita.yaml before running RITA\n"
			fi
		else
			printf "[+] MongoDB is Installed\n"
			printf "[-] Please enter MongoDB connection string (default is \"mongodb://localhost:27017\" press enter to keep default): "
			read mongodb_conn_str
			if [ "$mongodb_conn_str" == "" ]; then
				mongodb_conn_str='mongodb://localhost:27017'
			fi
  	fi
	fi

	( # Build RITA
		# Ensure go dep is installed
		wget -q -O $GOPATH/bin/dep https://github.com/golang/dep/releases/download/v0.3.2/dep-linux-amd64
		chmod +x $GOPATH/bin/dep

		mkdir -p $GOPATH/src/github.com/activecm/rita
		# Get the install script's directory in case it's run from elsewhere
		cp -R "$(dirname "$(realpath ${0})")/." $GOPATH/src/github.com/activecm/rita/
		cd $GOPATH/src/github.com/activecm/rita
		make install > /dev/null
		# Allow any user to execute rita
		chmod 755 $GOPATH/bin/rita
	)	& __load "[+] Installing RITA"


	( # Install the base configuration files
		mkdir -p $_CONFIG_PATH
		cd $GOPATH/src/github.com/activecm/rita
		cp ./LICENSE $_CONFIG_PATH/LICENSE
		cp ./etc/rita.yaml $_CONFIG_PATH/config.yaml
		cp ./etc/tables.yaml $_CONFIG_PATH/tables.yaml
		touch $_CONFIG_PATH/safebrowsing
		chmod 755 $_CONFIG_PATH
		# All users can read and write rita's config file
		chmod 666 $_CONFIG_PATH/config.yaml
		chmod 666 $_CONFIG_PATH/safebrowsing
	) & __load "[+] Installing config files to $_CONFIG_PATH"

	__mod_config $bro_log_path $mongodb_conn_str

#We should auto source these and start broctl and mongodb
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
	_INSTALL_BRO=true
	_INSTALL_MONGO=true
	_INSTALL_PREFIX=/opt/rita
	_CONFIG_PATH=/etc/rita
	_INSTALL_RITA=true
	_UNINSTALL_RITA=false

	# Parse through command args
	while [[ $# -gt 0 ]]; do
		case $1 in
			-h|--help)
				# Display help and exit
				__help
				exit 0
				;;
			-u|--uninstall)
				_UNINSTALL_RITA=true
				_INSTALL_RITA=false
				_INSTALL_BRO=false
				_INSTALL_MONGO=false
				;;
			--disable-bro)
				_INSTALL_BRO=false
				;;
			--disable-mongo)
				_INSTALL_MONGO=false
				;;
			--prefix)
				shift
				_INSTALL_PREFIX="$1"
				;;
			*)
			;;
	  esac
	  shift
	done

	# Check to see if the user has permission to install RITA
	if __checkPermissions
	then
		if [[ "${_UNINSTALL_RITA}" = "true" ]]
		then
			__uninstall
			exit 0
		fi
		if [[ "${_INSTALL_RITA}" = "true" ]]
		then
			__install
			exit 0
		fi
	else
		printf "You do NOT have permission install RITA\n\n"
	fi


}

__entry "${@:-}"
