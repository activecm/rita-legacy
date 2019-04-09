#!/usr/bin/env bash
#
# RITA is brought to you by Active CounterMeasures.
# activecountermeasures.com

# CONSTANTS
_RITA_VERSION="v3.0.0"
_MONGO_VERSION="3.6"
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

# ENTRYPOINT
__entry() {
	_REINSTALL_RITA=false

	# Optional Dependencies
	_INSTALL_BRO=true
	_INSTALL_MONGO=true
	_INSTALL_RITA=true

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
			--disable-rita)
				_INSTALL_RITA=false
				;;
			*)
			;;
		esac
		shift
	done

	if [ "$_INSTALL_BRO" = "false" -a "$_INSTALL_MONGO" = "false" -a "$_INSTALL_RITA" = "false" ]; then
		echo "No packages were selected for installation, exiting."
		exit 0
	fi

	if ! [ $(id -u) = 0 ]; then
		echo "You do not have permissions to install RITA!"
		exit 1
	fi

	_BIN_PATH="$_INSTALL_PREFIX/bin"

	if [ "$_INSTALL_RITA" = "true" ] && __installation_exist && [ "$_REINSTALL_RITA" != "true" ]; then
		printf "$_IMPORTANT RITA is already installed.\n"
		printf "$_QUESTION Would you like to re-install? [y/N] "
		read
		if [[ ! $REPLY =~ ^[Yy]$ ]]; then
			exit 0
		fi
		_REINSTALL_RITA="true"
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
	__gather_mongo

	# Explain the installer's actions
	__explain

	if [ "$_INSTALL_BRO" = "true" ]; then
		if [ "$_BRO_INSTALLED" = "false" ]; then
			__load "$_ITEM Installing Bro IDS" __install_bro
		else
			printf "$_ITEM Bro IDS is already installed \n"
		fi

		#Unconditionally installed whether this is a new install or an upgrade
		#Install this before calling __configure_bro so the modules are in place when "broctl deploy" restarts bro
		__install_ja3
		__enable_ssl_certificate_logging

		if [ "$_BRO_INSTALLED" = "true" ]; then
			__configure_bro
		fi

		if [ "$_BRO_IN_PATH" = "false" ]; then
			__add_bro_to_path
		fi
	fi

	if [ "$_INSTALL_MONGO" = "true" ]; then
		if [ "$_MONGO_INSTALLED" = "false" ]; then
			__load "$_ITEM Installing MongoDB" __install_mongodb
		elif ! __satisfies_version "$_MONGO_INSTALLED_VERSION" "$_MONGO_VERSION" ; then
			__load "$_ITEM Updating MongoDB" __install_mongodb
			# Need to also install all the components of the mongodb-org metapackage for Ubuntu
			__install_packages mongodb-org-mongos mongodb-org-server mongodb-org-shell mongodb-org-tools
		else
			printf "$_ITEM MongoDB is already installed \n"
		fi

		if [ "$_MONGO_INSTALLED" = "true" ]; then
			__configure_mongodb
		fi
	fi

	if [ "$_INSTALL_RITA" = "true" ]; then
		__load "$_ITEM Installing RITA" __install_rita
	fi

	if [ "$_REINSTALL_RITA" = "true" ]; then
		printf "$_IMPORTANT $_RITA_CONFIG_FILE may need to be updated for this version of RITA. \n"
		printf "$_IMPORTANT A default config file has been created at $_RITA_REINSTALL_CONFIG_FILE. \n"
		printf "$_IMPORTANT \"rita test-config\" may be used to troubleshoot configuration issues. \n \n"
	fi

	printf "$_IMPORTANT To finish the installation, reload the system profile with \n"
	printf "$_IMPORTANT 'source /etc/profile'.\n"

	__title
	printf "Thank you for installing RITA! Happy hunting! \n"
}

__install_installer_deps() {
	printf "$_ITEM In order to run the installer, several basic packages must be installed. \n"

	# Update package cache
	__load "$_SUBITEM Updating packages" __freshen_packages

	for pkg in curl coreutils lsb-release yum-utils; do
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
			# Workaround for https://github.com/activecm/rita/issues/189
			# Replace the download.opensuse.org link with downloadcontent.opensuse.org link
			# https://www.linuxquestions.org/questions/linux-general-1/yum-update-failed-because-of-timeout-4175625075/#post5828487
			cat '/etc/yum.repos.d/network:bro.repo' | sed -e 's/download\.opensuse\.org/downloadcontent.opensuse.org/g' | tee '/etc/yum.repos.d/network:bro.repo.tmp' >/dev/null && mv -f '/etc/yum.repos.d/network:bro.repo.tmp' '/etc/yum.repos.d/network:bro.repo'
			;;
	esac
	__install_packages bro broctl
	chmod 2755 /opt/bro/logs
	_BRO_PKG_INSTALLED=true
	_BRO_INSTALLED=true
	_BRO_PATH="/opt/bro/bin"
}

__install_ja3() {
	local_path=$_BRO_PATH/../share/bro/site/

	sudo mkdir -p $local_path/ja3/

	for one_file in __load__.bro intel_ja3.bro ja3.bro ja3s.bro ; do
		if [ ! -e $local_path/ja3/$one_file ]; then
			sudo curl -sSL "https://raw.githubusercontent.com/salesforce/ja3/master/bro//$one_file" -o "$local_path/ja3/$one_file"
		fi
	done

	if ! grep -q '^[^#]*@load \./ja3' $local_path/local.bro ; then
		echo '' >>$local_path/local.bro
		echo '#Load ja3 support libraries' >>$local_path/local.bro
		echo '@load ./ja3' >>$local_path/local.bro
	fi
}

__enable_ssl_certificate_logging() {
	local_path=$_BRO_PATH/../share/bro/site/

	sudo mkdir -p $local_path

	if ! grep -q '^[^#]*@load  *protocols/ssl/validate-certs' $local_path/local.bro ; then
		echo '' >>$local_path/local.bro
		echo '#Enable certificate validation' >>$local_path/local.bro
		echo '@load protocols/ssl/validate-certs' >>$local_path/local.bro
	fi

	if ! grep -q '^[^#]*@load  *policy/protocols/ssl/extract-certs-pem' $local_path/local.bro ; then
		echo '' >>$local_path/local.bro
		echo '#Log certificates' >>$local_path/local.bro
		echo '@load policy/protocols/ssl/extract-certs-pem' >>$local_path/local.bro
		echo 'redef SSL::extract_certs_pem = ALL_HOSTS;' >>$local_path/local.bro
		echo '' >>$local_path/local.bro
	fi
}

__configure_bro() {
	_BRO_CONFIGURED=false

	# Attempt to detect if Bro is already configured away from its defaults
	if [ -s "$_BRO_PATH/../etc/node.cfg" ] && grep -q '^type=worker' "$_BRO_PATH/../etc/node.cfg" ; then
		_BRO_CONFIGURED=true
	fi

	# Attempt to configure Bro interactively
	if [ "$_BRO_CONFIGURED" = "false" ]; then
		# Configure Bro
		tmpdir=`mktemp -d -q "/tmp/rita.XXXXXXXX" < /dev/null`
		if [ ! -d "$tmpdir" ]; then
			tmpdir=.
		fi
		curl -sSL "https://raw.githubusercontent.com/activecm/bro-install/master/gen-node-cfg.sh" -o "$tmpdir/gen-node-cfg.sh"
		curl -sSL "https://raw.githubusercontent.com/activecm/bro-install/master/node.cfg-template" -o "$tmpdir/node.cfg-template"
		chmod 755 "$tmpdir/gen-node-cfg.sh"
		if "$tmpdir/gen-node-cfg.sh" ; then
			_BRO_CONFIGURED=true
		fi
		# Clean up the files in case they ended up in the current directory
		rm -f "$tmpdir/gen-node-cfg.sh"
		rm -f "$tmpdir/node.cfg-template"
	fi

	if [ "$_BRO_CONFIGURED" = "true" ]; then
		printf "\n$_IMPORTANT Enabling Bro on startup.\n"
		# don't add a new line if broctl is already in cron
		if [ $(crontab -l 2>/dev/null | grep 'broctl cron' | wc -l) -eq 0 ]; then
			# Append an entry to an existing crontab
			# crontab -l will print to stderr and exit with code 1 if no crontab exists
			# Ignore stderr and force a successfull exit code to prevent an install error
			(crontab -l 2>/dev/null || true; echo "*/5 * * * * $_BRO_PATH/broctl cron") | crontab
		fi
		$_BRO_PATH/broctl cron enable > /dev/null
		printf "$_IMPORTANT Enabling Bro on startup process completed.\n"

		printf "$_IMPORTANT Starting Bro. \n"
		$_BRO_PATH/broctl deploy
	else
		printf "$_IMPORTANT Automatic Bro configuration failed. \n"
		printf "$_IMPORTANT Please edit /opt/bro/etc/node.cfg and run \n"
		printf "$_IMPORTANT 'sudo broctl deploy' to start Bro. \n"
		printf "$_IMPORTANT Pausing for 20 seconds before continuing. \n"
		sleep 20
	fi
}

__add_bro_to_path() {
	printf "$_SUBIMPORTANT Adding Bro IDS to the path in $_BRO_PATH_SCRIPT \n"
	echo "export PATH=\"\$PATH:$_BRO_PATH\"" | tee $_BRO_PATH_SCRIPT > /dev/null
	_BRO_PATH_SCRIPT_INSTALLED=true
	export PATH="$PATH:$_BRO_PATH"
	_BRO_IN_PATH=true
}

__install_mongodb() {
	case "$_OS" in
		Ubuntu)
			__add_deb_repo "deb [ arch=$(dpkg --print-architecture) ] http://repo.mongodb.org/apt/ubuntu $(lsb_release -cs)/mongodb-org/${_MONGO_VERSION} multiverse" \
				"mongodb-org-${_MONGO_VERSION}" \
				"https://www.mongodb.org/static/pgp/server-${_MONGO_VERSION}.asc"
			;;
		CentOS)
			if [ ! -s /etc/yum.repos.d/mongodb-org-${_MONGO_VERSION}.repo ]; then
				cat << EOF > /etc/yum.repos.d/mongodb-org-${_MONGO_VERSION}.repo
[mongodb-org-${_MONGO_VERSION}]
name=MongoDB Repository
baseurl=https://repo.mongodb.org/yum/redhat/\$releasever/mongodb-org/${_MONGO_VERSION}/x86_64/
gpgcheck=1
enabled=1
gpgkey=https://www.mongodb.org/static/pgp/server-${_MONGO_VERSION}.asc
EOF
			fi
			__freshen_packages
			;;
	esac
	__install_packages mongodb-org
	_MONGO_INSTALLED=true

}

__configure_mongodb() {
	printf "$_IMPORTANT Starting MongoDB and enabling on startup. \n"
	if [ $_OS = "Ubuntu" ]; then
		if [ $_OS_CODENAME = "trusty" ]; then
			# Ubuntu 14.04 uses Upstart for init
			# https://www.digitalocean.com/community/tutorials/how-to-configure-a-linux-service-to-start-automatically-after-a-crash-or-reboot-part-1-practical-examples#auto-start-checklist-for-upstart
			initctl stop mongod > /dev/null
			initctl start mongod > /dev/null
			_STOP_MONGO="sudo service mongod stop"
		else
			systemctl enable mongod.service > /dev/null
			systemctl daemon-reload > /dev/null
			systemctl start mongod > /dev/null
			_STOP_MONGO="sudo systemctl stop mongod"
		fi
	elif [ $_OS = "CentOS" ]; then
		systemctl enable mongod.service > /dev/null
		systemctl daemon-reload > /dev/null
		systemctl start mongod > /dev/null
		_STOP_MONGO="sudo systemctl stop mongod"
		#chkconfig mongod on > /dev/null
		#service mongod start > /dev/null
		#_STOP_MONGO="sudo service mongod stop"
	fi
	printf "$_IMPORTANT Starting MongoDB process completed.\n"

	printf "$_IMPORTANT You can access the MongoDB shell with 'mongo'. \n"
	printf "$_IMPORTANT If you need to stop MongoDB, \n"
	printf "$_IMPORTANT run '$_STOP_MONGO'. \n"
}

__install_rita() {
	_RITA_RELEASE_URL="https://github.com/activecm/rita/releases/download/$_RITA_VERSION"
	_RITA_REPO_URL="https://raw.githubusercontent.com/activecm/rita/$_RITA_VERSION"
	_RITA_BINARY_URL="$_RITA_RELEASE_URL/rita"
	_RITA_CONFIG_URL="$_RITA_REPO_URL/etc/rita.yaml"
	_RITA_LICENSE_URL="$_RITA_REPO_URL/LICENSE"

	_RITA_CONFIG_FILE="$_CONFIG_PATH/config.yaml"
	_RITA_REINSTALL_CONFIG_FILE="$_CONFIG_PATH/config.yaml.new"

	curl -sSL "$_RITA_BINARY_URL" -o "$_BIN_PATH/rita"
	chmod 755 "$_BIN_PATH/rita"

	mkdir -p "$_CONFIG_PATH"
	mkdir -p "$_VAR_PATH"
	chmod 755 "$_VAR_PATH"
	mkdir -p "$_VAR_PATH/logs"
	chmod 777 "$_VAR_PATH/logs"

	curl -sSL "$_RITA_LICENSE_URL" -o "$_CONFIG_PATH/LICENSE"

	if [ "$_REINSTALL_RITA" = "true" ]; then
		# Don't overwrite existing config
		curl -sSL "$_RITA_CONFIG_URL" -o "$_RITA_REINSTALL_CONFIG_FILE"
		chmod 666 "$_RITA_REINSTALL_CONFIG_FILE"
	else
		curl -sSL "$_RITA_CONFIG_URL" -o "$_RITA_CONFIG_FILE"
		chmod 666 "$_RITA_CONFIG_FILE"
	fi

	# All users can read and write rita's config file
	chmod 755 "$_CONFIG_PATH"

	mkdir -p /etc/bash_completion.d/
	curl -sSL "https://raw.githubusercontent.com/urfave/cli/master/autocomplete/bash_autocomplete" -o "/etc/bash_completion.d/rita"
}

# INFORMATION GATHERING

__gather_OS() {
	_OS="$(lsb_release -is)"
	_OS_CODENAME="$(lsb_release -cs)"
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
		_MONGO_INSTALLED_VERSION="$(__package_version mongodb-org)"
	fi
}

# USER EXPERIENCE

__explain() {
	printf "$_ITEM This installer will: \n"
	if [ $_BRO_INSTALLED = "false" -a $_INSTALL_BRO = "true" ]; then
		printf "$_SUBITEM Install Bro IDS to /opt/bro \n"
	fi
	if [ $_MONGO_INSTALLED = "false" -a $_INSTALL_MONGO = "true" ]; then
		printf "$_SUBITEM Install MongoDB \n"
	fi
	if ( ! __installation_exist ) && [ $_INSTALL_RITA = "true" ]; then
		printf "$_SUBITEM Install RITA to $_BIN_PATH/rita \n"
		printf "$_SUBITEM Create a runtime directory for RITA in $_VAR_PATH \n"
		printf "$_SUBITEM Create a configuration directory for RITA in $_CONFIG_PATH \n"
	fi
	sleep 5s
}

__title() {
	echo \
"
 _ \ _ _| __ __|  \\
   /   |     |   _ \\
_|_\ ___|   _| _/  _\\  $_RITA_VERSION

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
		# yum -> apt
		if [ $_PKG_MGR -eq 1 ]; then
			case "$pkg" in
				"yum-utils")
					# required for yum-config-manager
					# Ubuntu equivalent is apt-key which is already installed
					shift
					continue
					;;
			esac
		# apt -> yum
		elif [ $_PKG_MGR -eq 2 ]; then
			case "$pkg" in
				"lsb-release")
					pkg="redhat-lsb-core"
					;;
				"realpath")
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
		apt-get -qq update > /dev/null 2>&1 || /bin/true
	elif [ $_PKG_MGR -eq 2 ]; then #yum
		yum -q makecache > /dev/null 2>&1 || /bin/true
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

__package_version() {
	if ! __package_installed "$1"; then
		echo ""
	fi

	if [ $_PKG_MGR -eq 1 ]; then # apt
		echo $(dpkg-query -W -f '${Version}' "$1")
	elif [ $_PKG_MGR -eq 2 ]; then # yum and dnf
		echo $(rpm -qa --queryformat='%{VERSION}' "$1")
	fi
}

# Compares two version strings to determine if the first is
# less than or equal to the second. Version strings are expected
# to be in the form: 1.2.3, 1.2, or 1
__satisfies_version() {
	local installed="$1"
	local desired="$2"

	# Break apart version strings like 1.2.3 into major.minor.patch
	local installed_major="$(echo $installed | cut -d'.' -f1)"
	local installed_minor="$(echo $installed | cut -d'.' -f2)"
	local installed_patch="$(echo $installed | cut -d'.' -f3)"
	local desired_major="$(echo $desired | cut -d'.' -f1)"
	local desired_minor="$(echo $desired | cut -d'.' -f2)"
	local desired_patch="$(echo $desired | cut -d'.' -f3)"

	# Set any empty values to 0
	if [ -z "$installed_major" ]; then installed_major=0; fi
	if [ -z "$installed_minor" ]; then installed_minor=0; fi
	if [ -z "$installed_patch" ]; then installed_patch=0; fi
	if [ -z "$desired_major" ]; then desired_major=0; fi
	if [ -z "$desired_minor" ]; then desired_minor=0; fi
	if [ -z "$desired_patch" ]; then desired_patch=0; fi

	if [ "$installed_major" -lt "$desired_major" ]; then
		false; return
	elif [ "$installed_major" -gt "$desired_major" ]; then
		true; return
	fi
	# else major versions are equal and we need to check minor
	if [ "$installed_minor" -lt "$desired_minor" ]; then
		false; return
	elif [ "$installed_minor" -gt "$desired_minor" ]; then
		true; return
	fi
	# else minor versions are equal and we need to check patch
	if [ "$installed_patch" -lt "$desired_patch" ]; then
		false; return
	elif [ "$installed_patch" -gt "$desired_patch" ]; then
		true; return
	else
		# installed version is exactly equal to desired version
		true; return
	fi
}

__add_deb_repo() {
	if [ ! -s "/etc/apt/sources.list.d/$2.list" ]; then
		if [ ! -z "$3" ]; then
			curl -s -L "$3" | apt-key add - > /dev/null 2>&1
		fi
		echo "$1" | tee "/etc/apt/sources.list.d/$2.list" > /dev/null
		__freshen_packages
	fi
}

__add_rpm_repo() {
	yum-config-manager -q --add-repo=$1 > /dev/null 2>&1
	__freshen_packages
}

# ENTRYPOINT CALL
__entry "${@:-}"
