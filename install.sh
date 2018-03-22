#!/usr/bin/env bash
#
# RITA is brought to you by Active CounterMeasures.
# activecountermeasures.com

# CONSTANTS
_NAME=$(basename "${0}")
_FAILED="\e[91mFAILED\e[0m"
_SUCCESS="\e[92mSUCCESS\e[0m"
_ITEM="[-]"
_IMPORTANT="[!]"
_QUESTION="[?]"
_SUBITEM="\t$_ITEM"
_SUBIMPORTANT="\t$_IMPORTANT"
_SUBQUESTION="\t$_QUESTION"


# ERROR HANDLING
__err() {
	printf "\n$_IMPORTANT Installation $_FAILED on line $1.\n"
	exit 1
}

__int() {
	printf "\n$_IMPORTANT Installation \e[91mCANCELLED\e[0m.\n"
	exit 1
}

trap '__err $LINENO' ERR
trap '__int' INT

set -o errexit
set -o errtrace
set -o pipefail


# PERMISSIONS GADGET
# The user must run the build process, but root must install
# software. In order to make sure the appropriate users
# take the right actions, we call sudo in the script itself.

# Prevent user running sudo themselves
if [ ! -z ${SUDO_USER+x} ]; then
	printf "Please run the RITA installer without sudo.\n"
	exit 1
fi

# Root is running the script without sudo
if [ "$EUID" = "0" ]; then
	_ELEVATE=""
else
	printf "$_IMPORTANT The RITA installer requires root privileges for some tasks. \n"
	printf "$_IMPORTANT \"sudo\" will be used when necessary. \n"
	_SUDO="$(type -fp sudo)"
	if [ -z $_SUDO ]; then
		printf "\"sudo\" was not found on the system. Please log in as root \n"
		printf "before running the installer, or install \"sudo\". \n"
		exit 1
	fi
	$_SUDO -v
	if [ $? -ne 0 ]; then
		printf "The installer was unable to elevate privileges using \"sudo\". \n"
		printf "Please make sure your account has \"sudo\" privileges. \n"
	fi
	# _ELEVATE is separate from _SUDO since environment variables may need
	# to be passed
	_ELEVATE="$_SUDO"
fi

# ENTRYPOINT
__entry() {
	_REINSTALL_RITA=false

	# Optional Dependencies
	_INSTALL_BRO=true
	_INSTALL_MONGO=true

	# Install locations
	_INSTALL_PREFIX=/usr/local
	_CONFIG_PATH=/etc/rita
	_VAR_PATH=/var/lib/rita

	# FOR an OPT style installation
	# NOTE: RITA itself must be changed to agree with the
	# _CONFIG_PATH and _VAR_PATH
	# _INSTALL_PREFIX=/opt/rita
	# _CONFIG_PATH=/etc/opt/rita
	# _VAR_PATH=/var/opt/rita

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
				_INSTALL_BRO=false
				_INSTALL_MONGO=false
				;;
			--disable-bro)
				_INSTALL_BRO=false
				;;
			--disable-mongo)
				_INSTALL_MONGO=false
				;;
			-v|--version)
				shift
				_RITA_VERSION="$1"
				;;
			*)
			;;
	  esac
	  shift
	done

	_BIN_PATH="$_INSTALL_PREFIX/bin"

	if __installation_exist && [ "$_REINSTALL_RITA" != "true" ]; then
		printf "$_IMPORTANT RITA is already installed.\n"
		printf "$_QUESTION Would you like to re-install? [y/N] "
		read
		if [[ $REPLY =~ ^[Nn]$ ]]; then
			exit 0
		fi
	fi

	__install
}

__installation_exist() {
	[ -f "$_BIN_PATH/rita" -o -d "$_CONFIG_PATH" -o -d "$_VAR_PATH" ]
}

__install() {
	__title
	# Gather enough information to download installer dependencies
	__gather_pkg_mgr

	# Install installer dependencies
	__install_installer_deps

	# Get system information
	__gather_OS
	__gather_bro
	__gather_go
	__gather_mongo

	# Explain the installer's actions
	__explain

	if [ "$_INSTALL_BRO" = "true" ]; then
		if [ "$_BRO_INSTALLED" = "false" ]; then
			__load "$_ITEM Installing Bro IDS" __install_bro
		else
			printf "$_ITEM Bro IDS is already installed \n"
		fi

		if [ "$_BRO_IN_PATH" = "false" ]; then
			__add_bro_to_path
		fi
	fi

	# Always install Go
	if [ "$_GO_OUT_OF_DATE" = "true" ]; then
		printf "$_IMPORTANT WARNING: An old version of Go has been detected on this system. \n"
		printf "$_IMPORTANT RITA has only been tested with Go >= 1.7. Check if the installed \n"
		printf "$_IMPORTANT version of Go is up to date with 'go version'. If it is out of date \n"
		printf "$_IMPORTANT you may remove the old version of Go and let this installer install \n"
		printf "$_IMPORTANT a more recente version. \n"
		sleep 10s
	fi

	if [ "$_GO_INSTALLED" = "false" ]; then
		__load "$_ITEM Installing Go" __install_go
	else
		printf "$_ITEM Go is already installed \n"
	fi

	if [ "$_GO_IN_PATH" = "false" ]; then
		__add_go_to_path
	fi

	if [ "$_GOPATH_EXISTS" = "false" ]; then
		__create_go_path
	else
		printf "$_SUBITEM Found GOPATH at $GOPATH \n"
		# Add the bin folder of the $GOPATH
		# It may already be in the path, but oh well, better to be safe than sorry
		export PATH=$PATH:$GOPATH/bin
	fi

	if [ $_INSTALL_MONGO = "true" ]; then
		if [ $_MONGO_INSTALLED = "false" ]; then
			__load "$_ITEM Installing MongoDB" __install_mongodb
		else
			printf "$_ITEM MongoDB is already installed \n"
		fi
	fi

	__load "$_ITEM Installing RITA" __build_rita && __install_rita

	printf "$_IMPORTANT To finish the installtion, reload the system profile and \n"
	printf "$_IMPORTANT user profile with 'source /etc/profile' and 'source ~/.profile'. \n"
	printf "$_IMPORTANT Additionally, you may want to configure Bro and run 'sudo broctl deploy'. \n"
	printf "$_IMPORTANT Finally, start MongoDB with 'sudo systemctl start mongod'. You can \n"
	printf "$_IMPORTANT access the MongoDB shell with 'mongo'. If, at any time, you need \n"
	printf "$_IMPORTANT to stop MongoDB, run 'sudo systemctl stop mongod'. \n"

	__title
	printf "Thank you for installing RITA! Happy hunting! \n"
}

__install_installer_deps() {
	printf "$_ITEM In order to run the installer, several basic packages must be installed. \n"

	# Update package cache
	__load "$_SUBITEM Updating packages" __freshen_packages

	for pkg in git curl make coreutils lsb-release; do
		__load "$_SUBITEM Ensuring $pkg is installed" __install_packages $pkg
	done
}

__install_bro() {
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
	$_ELEVATE chmod 2755 /opt/bro/logs
	_BRO_PKG_INSTALLED=true
	_BRO_PATH="/opt/bro/bin"
}

__add_bro_to_path() {
	printf "$_SUBQUESTION Would you like to add Bro IDS to the PATH? [Y/n] "
	read
	if [[ ! $REPLY =~ ^[Nn]$ ]]; then
		printf "$_SUBIMPORTANT Adding Bro IDS to the path in $_BRO_PATH_SCRIPT \n"
		echo "export PATH=\"\$PATH:$_BRO_PATH\"" | $_ELEVATE tee $_BRO_PATH_SCRIPT > /dev/null
		_BRO_PATH_SCRIPT_INSTALLED=true
		export PATH="$PATH:$_BRO_PATH"
		_BRO_IN_PATH=true
	fi
}

__install_go() {
		curl -s -o /tmp/golang.tar.gz https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz
		$_ELEVATE tar -zxf /tmp/golang.tar.gz -C /usr/local/
		rm /tmp/golang.tar.gz
		_GO_INSTALLED_STD=true
		_GO_INSTALLED=true
		_GO_PATH="/usr/local/go/bin"
}

__add_go_to_path() {
		printf "$_SUBIMPORTANT Adding Go to the path in $_GO_PATH_SCRIPT \n"
		echo "export PATH=\"\$PATH:$_GO_PATH\"" | $_ELEVATE tee $_GO_PATH_SCRIPT > /dev/null
		_GO_PATH_SCRIPT_INSTALLED=true
		export PATH="$PATH:$_GO_PATH"
		_GO_IN_PATH=true
}

__create_go_path() {
	printf "$_SUBIMPORTANT Go requires a per-user workspace (GOPATH) in order to build software. \n"

	printf "$_SUBQUESTION Select a GOPATH [$HOME/go]: "
	read
	if [ -n "$REPLY" ]; then
		export GOPATH="$REPLY"
	else
		export GOPATH="$HOME/go"
	fi

	printf "$_SUBIMPORTANT Creating a GOPATH at $GOPATH \n"
	mkdir -p "$GOPATH/"{src,pkg,bin}
	_GOPATH_EXISTS=true

	export PATH="$PATH:$GOPATH/bin"

	printf "$_SUBIMPORTANT Adding your GOPATH to $_GOPATH_PATH_SCRIPT \n"
	echo "export GOPATH=\"$GOPATH\"" > "$_GOPATH_PATH_SCRIPT"
	echo "export PATH=\"\$PATH:\$GOPATH/bin\"" >> "$_GOPATH_PATH_SCRIPT"
	_GOPATH_PATH_SCRIPT_INSTALLED=true

	printf "$_SUBIMPORTANT Adding $_GOPATH_PATH_SCRIPT to $HOME/.profile \n"
	echo "source \"$_GOPATH_PATH_SCRIPT\"" >> "$HOME/.profile"
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
	_MONGO_INSTALLED=true
}

__build_rita() {
	curl -L -s -o "$GOPATH/bin/dep" https://github.com/golang/dep/releases/download/v0.3.2/dep-linux-amd64
	chmod +x "$GOPATH/bin/dep"

	export _RITA_SRC_DIR="$GOPATH/src/github.com/activecm/rita"
	mkdir -p "$_RITA_SRC_DIR"

	# Get the code from git since the build process is dependent on git
	git clone http://github.com/activecm/rita "$_RITA_SRC_DIR" > /dev/null 2>&1

	local old_dir="$PWD"
	cd "$_RITA_SRC_DIR"
	if [ -n "${_RITA_VERSION+x}" ]; then
		git checkout $_RITA_VERSION > /dev/null 2>&1
	fi
	make > /dev/null
	cd "$old_dir"
}

__install_rita() {
	$_ELEVATE mkdir -p "$_CONFIG_PATH"
	#$_ELEVATE mkdir -p "$_VAR_PATH"
	$_ELEVATE mkdir -p "$_VAR_PATH/logs"

	$_ELEVATE mv -f "$_RITA_SRC_DIR/rita" "$_BIN_PATH/rita"
	$_ELEVATE chown root:root "$_BIN_PATH/rita"
	$_ELEVATE chmod 755 "$_BIN_PATH/rita"

	$_ELEVATE cp -f "$_RITA_SRC_DIR/LICENSE" "$_CONFIG_PATH/LICENSE"
	if [ -f "$_CONFIG_PATH/config.yaml" ]; then
		printf "$_SUBITEM Backing up your current RITA config: $_CONFIG_PATH/config.yaml -> $_CONFIG_PATH/config.yaml.old \n"
		$_ELEVATE mv -f "$_CONFIG_PATH/config.yaml" "$_CONFIG_PATH/config.yaml.old"
	fi
	$_ELEVATE cp -f "$_RITA_SRC_DIR/etc/rita.yaml" "$_CONFIG_PATH/config.yaml"
	$_ELEVATE cp -f "$_RITA_SRC_DIR/etc/tables.yaml" "$_CONFIG_PATH/tables.yaml"

	# All users can read and write rita's config file
	$_ELEVATE chmod 755 "$_CONFIG_PATH"
	$_ELEVATE chmod 666 "$_CONFIG_PATH/config.yaml"

	$_ELEVATE touch "$_VAR_PATH/safebrowsing"
	$_ELEVATE chmod 755 "$_VAR_PATH"
	$_ELEVATE chmod 666 "$_VAR_PATH/safebrowsing"
}

# INFORMATION GATHERING

__gather_OS() {
	_OS="$(lsb_release -is)"
	if [ "$_OS" != "Ubuntu" -a "$_OS" != "CentOS" ]; then
		printf "$_ITEM This installer supports Ubuntu and CentOS. \n"
		printf "$_IMPORTANT Your operating system is unsupported."
		exit 1
	fi
}

__gather_pkg_mgr() {
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
		printf "$_ITEM This installer supports package management via apt and yum. \n"
		printf "$_IMPORTANT A supported package manager was not found. \n"
		exit 1
	fi
}

__gather_go() {
	_GO_PATH=""
	_GO_INSTALLED_STD=false
	if [ -f "/usr/local/go/bin/go" ]; then
		_GO_INSTALLED_STD=true
		_GO_PATH="/usr/local/go/bin"
	fi

	_GO_INSTALLED_NON_STD=false
	if [ -n "$GOROOT" -a -f "$GOROOT/bin/go" ]; then
		_GO_INSTALLED_NON_STD=true
		_GO_PATH="$GOROOT/bin"
	fi

	_GO_INSTALLED=false
	if [ $_GO_INSTALLED_STD = "true" -o $_GO_INSTALLED_NON_STD = "true" ]; then
		_GO_INSTALLED=true
	fi

	_GO_OUT_OF_DATE=false
	if [ $_GO_INSTALLED = "true" ]; then
		case `$_GO_PATH/go version | awk '{print $3}'` in
		go1|go1.2*|go1.3*|go1.4*|go1.5*|go1.6*|"")
			_GO_OUT_OF_DATE=true
			;;
		esac
	fi

	_GO_PATH_SCRIPT="/etc/profile.d/go-path.sh"
	_GO_PATH_SCRIPT_INSTALLED=false

	if [ -f "$_GO_PATH_SCRIPT" ]; then
		source "$_GO_PATH_SCRIPT"
		_GO_PATH_SCRIPT_INSTALLED=true
	fi

	_GO_IN_PATH=false
	if [ -n "$(type -fp go)" ]; then
		_GO_IN_PATH=true
	fi

	_GOPATH_PATH_SCRIPT="$HOME/.gopath-path.sh"
	_GOPATH_PATH_SCRIPT_INSTALLED=false

	if [ -f "$_GOPATH_PATH_SCRIPT" ]; then
		source "$_GOPATH_PATH_SCRIPT"
		_GOPATH_PATH_SCRIPT_INSTALLED=true
	fi

	_GOPATH_EXISTS=false
	if [ -n "$GOPATH" ]; then
		_GOPATH_EXISTS=true
	fi

}

__gather_bro() {
	_BRO_PATH=""
	_BRO_PKG_INSTALLED=false
	if __package_installed bro; then
		_BRO_PKG_INSTALLED=true
		_BRO_PATH="/opt/bro/bin"
	fi

	_BRO_ONION_INSTALLED=false
	if __package_installed securityonion-bro; then
		_BRO_ONION_INSTALLED=true
		_BRO_PATH="/opt/bro/bin"
	fi

	_BRO_SOURCE_INSTALLED=false
	if [ -f "/usr/local/bro/bin/bro" ]; then
		_BRO_SOURCE_INSTALLED=true
		_BRO_PATH="/usr/local/bro/bin"
	fi

	_BRO_INSTALLED=false
	if [ $_BRO_PKG_INSTALLED = "true" -o $_BRO_ONION_INSTALLED = "true" -o $_BRO_SOURCE_INSTALLED = "true" ]; then
		_BRO_INSTALLED=true
	fi

	_BRO_PATH_SCRIPT="/etc/profile.d/bro-path.sh"
	_BRO_PATH_SCRIPT_INSTALLED=false

	if [ -f "$_BRO_PATH_SCRIPT" ]; then
		source "$_BRO_PATH_SCRIPT"
		_BRO_PATH_SCRIPT_INSTALLED=true
	fi

	_BRO_IN_PATH=false
	if [ -n "$(type -pf bro)" ]; then
		_BRO_IN_PATH=true
	fi
}

__gather_mongo() {
	_MONGO_INSTALLED=false
	if __package_installed mongodb-org; then
		_MONGO_INSTALLED=true
	fi
}

# USER EXPERIENCE

__explain() {
	printf "$_ITEM This installer will: \n"
	if [ $_BRO_INSTALLED = "false" -a $_INSTALL_BRO = "true" ]; then
		printf "$_SUBITEM Install Bro IDS to /opt/bro \n"
	fi
	if [ $_GO_INSTALLED = "false" ]; then
		printf "$_SUBITEM Install Go to /usr/local/go \n"
	fi
	if [ $_GOPATH_EXISTS = "false" ]; then
		printf "$_SUBITEM Create a Go build environment (GOPATH) in $HOME/go \n"
	fi
	if [ $_MONGO_INSTALLED = "false" -a $_INSTALL_MONGO = "true" ]; then
		printf "$_SUBITEM Install MongoDB \n"
	fi
	printf "$_SUBITEM Install RITA to $_BIN_PATH/rita \n"
	printf "$_SUBITEM Create a runtime directory for RITA in $_VAR_PATH \n"
	printf "$_SUBITEM Create a configuration directory for RITA in $_CONFIG_PATH \n"
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
	-v|--version <version>		Specify the version tag of RITA to install instead of master.
	--disable-bro			Disable automatic installation of Bro IDS.
	--disable-mongo			Disable automatic installation of MongoDB.
HEREDOC
}

__load() {
  local loadingText=$1
	printf "$loadingText...\r"
	shift
	eval "$@"
	echo -ne "\r\033[K"
	printf "$loadingText... $_SUCCESS\n"
}

# PACKAGE MANAGEMENT
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
		eval $_ELEVATE $_PKG_INSTALL $pkg >/dev/null 2>&1
		shift
	done
}

__freshen_packages() {
	if [ $_PKG_MGR -eq 1 ]; then   #apt
		$_ELEVATE apt-get -qq update > /dev/null 2>&1
	elif [ $_PKG_MGR -eq 2 ]; then #yum
		$_ELEVATE yum -q makecache > /dev/null 2>&1
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
			curl -s -L "$3" | $_ELEVATE apt-key add - > /dev/null 2>&1
		fi
		echo "$1" | $_ELEVATE tee "/etc/apt/sources.list.d/$2.list" > /dev/null
		__freshen_packages
	fi
}

__add_rpm_repo() {
	$_ELEVATE yum-config-manager -q --add-repo=$1 > /dev/null 2>&1
}

# ENTRYPOINT CALL
__entry "${@:-}"
