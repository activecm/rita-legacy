#!/usr/bin/env bash
#
# RITA is brought to you by Offensive CounterMeasures.
# offensivecountermeasures.com

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
This script automatically installs Real Intelligence Threat Analyitics (RITA)
along with necessary dependencies, including Bro IDS and MongoDB.

Usage:
	${_NAME} [<arguments>]

Options:
	-h|--help			Show this help message.
	-r|--reinstall			Force reinstalling RITA.
	-b|--build			Force building RITA from source code.
	--disable-bro			Disable automatic installation of Bro IDS.
	--disable-mongo			Disable automatic installation of MongoDB.

HEREDOC
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

__check_permissions() {
	[ `id -u` -eq 0 ]
}

__rita_installed() {
	[[ -f $_INSTALL_PREFIX/bin/rita ]] \
	|| [[ ! -f $_CONFIG_PATH/rita.yaml ]] \
	|| [[ ! -f $_CONFIG_PATH/tables.yaml ]] \
	|| [[ ! -f $_CONFIG_PATH/LICENSE ]]
}

__set_pkgmgr() {
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
		_err
	fi
}

__set_os() {
	_OS="$(lsb_release -is)"
	if [ "$_OS" != "Ubuntu" -a "$_OS" != "CentOS" ]; then
		echo "Unsupported operating system"
		_err
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
	# Check if go is already installed
	if [ ! $(command -v go) ];	then
		if [ ! -x "/usr/local/go/bin/go" ]; then
			curl -s -o /tmp/golang.tar.gz https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz
			tar -zxf /tmp/golang.tar.gz -C /usr/local/
			rm /tmp/golang.tar.gz
		fi
		export PATH="$PATH:/usr/local/go/bin"
		echo 'export PATH=$PATH:/usr/local/go/bin' >> $HOME/.bashrc
	fi
}

__install_bro() {
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

	if [ ! $(command -v bro) ];	then
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

__install_build_env() {
	# TODO do this in a temporary directory so we don't leave go installed on the system
	(
		__install_packages git make

		__install_go
		__check_go_version

		_TMP_DIR=`mktemp -d -q "/tmp/rita.XXXXXXXX" </dev/null`
		if [[ ! -d $_TMP_DIR ]]
		then
		  # Fallback to $_INSTALL_PREFIX like before
		  _TMP_DIR=$_INSTALL_PREFIX
		fi

		# Override GOPATH to build RITA
		export GOPATH=$_INSTALL_PREFIX
		export PATH=$PATH:$GOPATH/bin

		mkdir -p $GOPATH/{src,pkg,bin}

		# Ensure dep is installed
		curl --silent --output $GOPATH/bin/dep https://github.com/golang/dep/releases/download/v0.3.2/dep-linux-amd64
		chmod +x $GOPATH/bin/dep
	) & __load "[+] Installing build environment"

	# Clean out existing RITA source
	rm -rf $GOPATH/src/github.com/ocmdev/rita
	mkdir -p $GOPATH/src/github.com/ocmdev/rita

	# Get RITA's source code
	# First check if the source is available locally
	if [[ -f ./rita.go ]]
	then
		# Copy source code from current directory to GOPATH
		# Suppress any error messages and always return success 
		# in case the installer is run from the GOPATH rita dir
		(cp -R . $GOPATH/src/github.com/ocmdev/rita 2> /dev/null || true) \
			& __load  "[+] Using locally available RITA source"
	else
		# TODO This will fail with current master branch. Make an argument to checkout a certain tag or branch.
		mkdir -p $GOPATH/src/github.com/ocmdev
		# Othwerwise clone the source code from Github
		(git clone https://github.com/ocmdev/rita $GOPATH/src/github.com/ocmdev/rita > /dev/null 2>&1) \
			& __load  "[+] Cloning RITA source from Github"
	fi
}

__build_rita() {
	(
		cd $GOPATH/src/github.com/ocmdev/rita
		make > /dev/null
	)
}

__install_rita() {
	mkdir -p $_CONFIG_PATH
	if [[ $_REINSTALL_RITA = "true" ]] && [[ -f $_CONFIG_PATH/config.yaml ]]
	then
		# TODO this overwrites the user config if run twice in a row
		(cp -f $_CONFIG_PATH/config.yaml $_CONFIG_PATH/config.yaml.old) \
			& __load "[+] Backing up configuration to $_CONFIG_PATH/config.yaml.old"
	fi
	(
		cp -f ./rita $_INSTALL_PREFIX/bin/rita
		chmod 755 $_INSTALL_PREFIX/bin/rita
		cp -f ./LICENSE $_CONFIG_PATH/LICENSE
		cp -f ./etc/rita.yaml $_CONFIG_PATH/config.yaml
		cp -f ./etc/tables.yaml $_CONFIG_PATH/tables.yaml
		touch $_CONFIG_PATH/safebrowsing
		chmod 755 $_CONFIG_PATH
		# All users can read and write rita's config file
		chmod 666 $_CONFIG_PATH/config.yaml
		chmod 666 $_CONFIG_PATH/safebrowsing
		) & __load "[+] Installing RITA binary and config files"
}

__parse_args() {
	_INSTALL_BRO=true
	_INSTALL_MONGO=true
	_INSTALL_PREFIX=/opt/rita
	_CONFIG_PATH=/etc/rita
	_REINSTALL_RITA=false
	_BUILD_RITA=false

	# Parse through command args
	while [[ $# -gt 0 ]]; do
		case $1 in
			-h|--help)
				# Display help and exit
				__help
				exit 0
				;;
			-r|--reinstall)
				_REINSTALL_RITA=true
				;;
			-b|--build)
				_BUILD_RITA=true
				;;
			--disable-bro) 
				_INSTALL_BRO=false
				;;
			--disable-mongo) 
				_INSTALL_MONGO=false
				;;
			# Note: prefix is purposely undocumented and should be used with care
			--prefix)
				shift
				# realpath makes sure it's an absolute path
				_INSTALL_PREFIX="$(realpath "$1")"
				;;
			*)
			;;
	  esac
	  shift
	done
}

# start point for installer
__entry() {	
	__parse_args "${@:-}"

	# Check to see if the user has permission to install RITA
	# TODO simply check if we have write permissions to _INSTALL_PREFIX
	if ! __check_permissions
	then
		printf "You do NOT have permission install RITA\n\n"
		exit 1
	fi

	if [[ $_REINSTALL_RITA != "true" ]] && __rita_installed
	then
		printf "[+] RITA is already installed in $_INSTALL_PREFIX.\n"
		read -p "[-] Would you like to re-install? [y/n] " -r
		if [[ $REPLY =~ ^[Yy]$ ]]
		then
			_REINSTALL_RITA=true
		else
			exit 1
		fi
	fi

	# Explain the scripts actions
	__explain

	# Figure out which package manager to use
	__set_pkgmgr

	# Update package sources
	__freshen_packages

	# Install "the basics"
	__install_packages curl coreutils realpath lsb-release & \
		__load "[+] Ensuring curl, coreutils, realpath, and lsb-release are installed"

	# Determine the OS, needs lsb-release
	__set_os

	if [[ $_INSTALL_BRO = "true" ]]
	then
		__install_bro & __load "[+] Installing Bro IDS"
	fi

	if [[ $_INSTALL_MONGO = "true" ]]
	then
		__install_mongodb & __load "[+] Installing MongoDB"
	fi

	# Go to the install script's directory in case this script is run from elsewhere
	cd "$(dirname "$(realpath $0)")"

	# Check if we should build from source 
	if [[ $_BUILD_RITA = "true" ]] \
		|| [[ ! -f ./rita ]] \
		|| [[ ! -f ./etc/rita.yaml ]] \
		|| [[ ! -f ./etc/tables.yaml ]] \
		|| [[ ! -f ./LICENSE ]]
	then
		if [[ $_BUILD_RITA = "false" ]]
		then
			printf "[+] Prebuilt RITA not available. Building from source.\n"
		fi
		__install_build_env
		__build_rita & __load "[+] Building RITA"
		# Compiled files will be here, so cd before installing
		cd $GOPATH/src/github.com/ocmdev/rita
	else
		printf "[+] Using prebuilt RITA binary.\n"
	fi

	__install_rita


	echo -e "
In order to finish the installation:
	1) Reload your bash config with 'source ~/.bashrc'. 
	2) Configure Bro and run 'sudo broctl deploy'. 
	3) Start the MongoDB service with 'sudo service mongod start'."

	__title
	printf "Thank you for installing RITA! "
	printf "Happy hunting\n"

}

__entry "${@:-}"
