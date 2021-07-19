#!/usr/bin/env bash
#
# RITA is brought to you by Active CounterMeasures.
# activecountermeasures.com

# CONSTANTS
_RITA_VERSION="v4.3.0"
_MONGO_VERSION="4.2"
_MONGO_MIN_UPDATE_VERSION="4.0"
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
    _INSTALL_ZEEK=true
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
                _INSTALL_ZEEK=false
                _INSTALL_MONGO=false
                ;;
            --disable-zeek|--disable-bro)
                _INSTALL_ZEEK=false
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

    if [ "$_INSTALL_ZEEK" = "false" -a "$_INSTALL_MONGO" = "false" -a "$_INSTALL_RITA" = "false" ]; then
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
        read -e
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
    __bro_installed
    __gather_zeek
    __gather_mongo

    # Explain the installer's actions
    __explain

    if [ "$_INSTALL_ZEEK" = "true" ]; then
        if [ "$_ZEEK_INSTALLED" = "false" ]; then
            __load "$_ITEM Installing Zeek IDS" __install_zeek
        else
            printf "$_ITEM Zeek IDS is already installed \n"
        fi

        #Unconditionally installed whether this is a new install or an upgrade
        #Install this before calling __configure_zeek so the modules are in place when "zeekctl deploy" restarts zeek
        __install_ja3
        __fix_inactivity_timeout
        __enable_ssl_certificate_logging

        if [ "$_ZEEK_INSTALLED" = "true" ]; then
            __configure_zeek
        fi

        if [ "$_ZEEK_IN_PATH" = "false" ]; then
            __add_zeek_to_path
        fi
    fi

    if [ "$_INSTALL_MONGO" = "true" ]; then
        if [ "$_MONGO_INSTALLED" = "false" ]; then
            __load "$_ITEM Installing MongoDB" __install_mongodb "$_MONGO_VERSION" 
        elif ! __satisfies_version "$_MONGO_INSTALLED_VERSION" "$_MONGO_VERSION" ; then

            # Check that the user wants to upgrade
            __mongo_upgrade_info

            # Check if the version is less than 4.0. If so, we need to update to 4.0
            # before going to 4.2
            if ! __satisfies_version "$_MONGO_INSTALLED_VERSION" "$_MONGO_MIN_UPDATE_VERSION"; then
                printf "$_ITEM Detected Mongo version less than $_MONGO_MIN_UPDATE_VERSION \n"
                __load "$_ITEM Performing intermediary update of MongoDB" __intermediary_update_mongodb
            fi

            # Need to stop mongo before updating otherwise we can end up with weird issues,
            # such as the console version not matching the server version
            systemctl is-active --quiet mongod && systemctl stop mongod

            __load "$_ITEM Updating MongoDB" __install_mongodb "$_MONGO_VERSION"

            # Need to also install all the components of the mongodb-org metapackage for Ubuntu
            __install_packages mongodb-org-mongos mongodb-org-server mongodb-org-shell mongodb-org-tools
        else
            printf "$_ITEM MongoDB is already installed \n"
        fi

        if [ "$_MONGO_INSTALLED" = "true" ]; then
            __configure_mongodb

            # Wait for service to come to life
            printf "$_ITEM Sleeping to give the Mongo service some time to fully start..."
            sleep 10
             
            # Set compatibility version in case we updated Mongo. It's fine to do this even if we didn't
            # update Mongo...it's just a bit cleaner to do it here to cut down on code redundancy and logic checks
            __load "$_ITEM Setting Mongo feature compatibility to $_MONGO_VERSION" __update_feature_compatibility "$_MONGO_VERSION"
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

__install_zeek() {

    if  [ "$_OS" == "Ubuntu" ] && [ "$_OS_CODENAME" == "xenial" ] ; then
        # The Zeek Project does not host a package repo for Ubuntu 16.04.
        # Use Security Onion's version of Zeek.
        sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 9373E47F9BF8216D23D32BBBE1E6759023F386C7 > /dev/null 2>&1
        __add_deb_repo "deb http://ppa.launchpad.net/securityonion/stable/ubuntu xenial main" \
        "securityonion-ubuntu-stable-xenial"
        __install_packages python securityonion-bro
        ln -s /opt/bro /opt/zeek
    else
        case "$_OS" in
            Ubuntu)
                __add_deb_repo "deb [ arch=$(dpkg --print-architecture) ] http://download.opensuse.org/repositories/security:/zeek/xUbuntu_$(lsb_release -rs)/ /" \
                "security:zeek" \
                "https://download.opensuse.org/repositories/security:/zeek/xUbuntu_$(lsb_release -rs)/Release.key"
                ;;
            CentOS|RedHatEnterprise|RedHatEnterpriseServer)
                __add_rpm_repo "https://download.opensuse.org/repositories/security:/zeek/CentOS_7/security:zeek.repo"
                ;;
        esac
        __install_packages zeek-lts
    fi


    if [ -d /opt/zeek/logs/ ]; then		#Standard directory for Zeek logs when installed by Rita
        chmod 2755 /opt/zeek/logs
    elif [ -d /var/log/zeek/ ]; then		#Standard directory for Zeek logs when installed by apt...
        mkdir -p /opt/zeek/logs/		#...and we move the log storage over to /opt/zeek/logs so we can have one place to mount external storage.
        chmod 2755 /opt/zeek/logs
        mv -f /var/log/zeek /var/log/zeek.orig
        cd /var/log
        ln -s /opt/zeek/logs zeek
        cd -
    fi
    _ZEEK_PKG_INSTALLED=true
    _ZEEK_INSTALLED=true
    _ZEEK_PATH="/opt/zeek/bin"
}

__install_ja3() {
    local_path="$_ZEEK_PATH/../share/zeek/site/"

    mkdir -p "$local_path/ja3/"

    for one_file in __load__.zeek intel_ja3.zeek ja3.zeek ja3s.zeek ; do
        if [ ! -e "$local_path/ja3/$one_file" ]; then
            curl -sSL "https://raw.githubusercontent.com/salesforce/ja3/133f2a128b873f9c40e4e65c2b9dc372a801cf24/zeek/$one_file" -o "$local_path/ja3/$one_file"
        fi
    done

    if ! grep -q '^[^#]*@load \./ja3' "$local_path/local.zeek" ; then
        echo '' >>"$local_path/local.zeek"
        echo '#Load ja3 support libraries' >>"$local_path/local.zeek"
        echo '@load ./ja3' >>"$local_path/local.zeek"
    fi
}

__fix_inactivity_timeout() {
    local_path="$_ZEEK_PATH/../share/zeek/site/"

    mkdir -p "$local_path"

    if ! grep -q '^[^#]*redef tcp_inactivity_timeout = 60 min;' "$local_path/local.zeek" ; then
        echo '' >>"$local_path/local.zeek"
        echo '#Extend inactivity timeout to collect lots of short connections' >>"$local_path/local.zeek"
        echo 'redef tcp_inactivity_timeout = 60 min;' >>"$local_path/local.zeek"
    fi
}

__enable_ssl_certificate_logging() {
    local_path="$_ZEEK_PATH/../share/zeek/site/"

    mkdir -p "$local_path"

    if ! grep -q '^[^#]*@load  *protocols/ssl/validate-certs' "$local_path/local.zeek" ; then
        echo '' >>"$local_path/local.zeek"
        echo '#Enable certificate validation' >>"$local_path/local.zeek"
        echo '@load protocols/ssl/validate-certs' >>"$local_path/local.zeek"
    fi

    if ! grep -q '^[^#]*@load  *policy/protocols/ssl/extract-certs-pem' "$local_path/local.zeek" ; then
        echo '' >>"$local_path/local.zeek"
        echo '#Log certificates' >>"$local_path/local.zeek"
        echo '@load policy/protocols/ssl/extract-certs-pem' >>"$local_path/local.zeek"
        echo 'redef SSL::extract_certs_pem = ALL_HOSTS;' >>"$local_path/local.zeek"
        echo '' >>"$local_path/local.zeek"
    fi
}

__configure_zeek() {
    _ZEEK_CONFIGURED=false

    # Attempt to detect if Zeek is already configured away from its defaults
    if [ -s "$_ZEEK_PATH/../etc/node.cfg" ] && grep -q '^type=worker' "$_ZEEK_PATH/../etc/node.cfg" ; then
        _ZEEK_CONFIGURED=true
    fi

    # Attempt to configure Zeek interactively
    if [ "$_ZEEK_CONFIGURED" = "false" ]; then
        # Configure Zeek
        tmpdir=`mktemp -d -q "$HOME/rita-install.XXXXXXXX" < /dev/null`
        if [ ! -d "$tmpdir" ] || findmnt -n -o options -T "$tmpdir" | grep -qE '(^|,)noexec($|,)'; then
            tmpdir=.
        fi
        curl -sSL "https://raw.githubusercontent.com/activecm/bro-install/master/gen-node-cfg.sh" -o "$tmpdir/gen-node-cfg.sh"
        curl -sSL "https://raw.githubusercontent.com/activecm/bro-install/master/node.cfg-template" -o "$tmpdir/node.cfg-template"
        chmod 755 "$tmpdir/gen-node-cfg.sh"
        if "$tmpdir/gen-node-cfg.sh" ; then
            _ZEEK_CONFIGURED=true
        fi
        # Clean up the files in case they ended up in the current directory
        rm -f "$tmpdir/gen-node-cfg.sh"
        rm -f "$tmpdir/node.cfg-template"
    fi

    if [ "$_ZEEK_CONFIGURED" = "true" ]; then
        printf "\n$_IMPORTANT Enabling Zeek on startup.\n"
        # don't add a new line if zeekctl is already in cron
        if [ $(crontab -l 2>/dev/null | grep 'zeekctl cron' | wc -l) -eq 0 ]; then
            # Append an entry to an existing crontab
            # crontab -l will print to stderr and exit with code 1 if no crontab exists
            # Ignore stderr and force a successfull exit code to prevent an install error
            (crontab -l 2>/dev/null || true; echo "*/5 * * * * $_ZEEK_PATH/zeekctl cron") | crontab
        fi
        $_ZEEK_PATH/zeekctl cron enable > /dev/null
        printf "$_IMPORTANT Enabling Zeek on startup process completed.\n"

        printf "$_IMPORTANT Starting Zeek. \n"
        $_ZEEK_PATH/zeekctl deploy
    else
        printf "$_IMPORTANT Automatic Zeek configuration failed. \n"
        printf "$_IMPORTANT Please edit /opt/zeek/etc/node.cfg and run \n"
        printf "$_IMPORTANT 'sudo zeekctl deploy' to start Zeek. \n"
        printf "$_IMPORTANT Pausing for 20 seconds before continuing. \n"
        sleep 20
    fi
}

__add_zeek_to_path() {
    printf "$_SUBIMPORTANT Adding Zeek IDS to the path in $_ZEEK_PATH_SCRIPT \n"
    echo "export PATH=\"\$PATH:$_ZEEK_PATH\"" | tee $_ZEEK_PATH_SCRIPT > /dev/null
    _ZEEK_PATH_SCRIPT_INSTALLED=true
    export PATH="$PATH:$_ZEEK_PATH"
    _ZEEK_IN_PATH=true
}

__mongo_upgrade_info() {
    printf "$_IMPORTANT Mongo is already installed and is version $_MONGO_INSTALLED_VERSION.\n"
    printf "$_IMPORTANT This will upgrade Mongo to version $_MONGO_VERSION.\n"
    printf "$_IMPORTANT Note that Mongo must be upgraded to $_MONGO_VERSION for RITA $_RITA_VERSION to work.\n"
    printf "$_IMPORTANT We suggest creating a backup of your data before upgrading (https://docs.mongodb.com/manual/tutorial/backup-and-restore-tools/).\n"
    printf "$_QUESTION Would you like to upgrade your Mongo instance [y/N] "
    read -e
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 0
    fi
}

__intermediary_update_mongodb() {
    # Need to stop mongo before updating otherwise we can end up with weird issues,
    # such as the console version not matching the server version
    systemctl is-active --quiet mongod && systemctl stop mongod

    __install_mongodb "$_MONGO_MIN_UPDATE_VERSION"

    # Need to also install all the components of the mongodb-org metapackage for Ubuntu
    __install_packages mongodb-org-mongos mongodb-org-server mongodb-org-shell mongodb-org-tools

    if [ "$_MONGO_INSTALLED" = "true" ]; then
        # Star and configure the service so that we can run the command to update feature compatibility
        __configure_mongodb

        # Wait for service to come to life
        printf "$_ITEM Sleeping to give the Mongo service some time to fully start..."
        sleep 10

        # Need to update feature compatibility to 4.0 otherwise things will break when we update
        # to 4.2
        __load "$_ITEM Setting feature compatibility in Mongo to $_MONGO_MIN_UPDATE_VERSION" __update_feature_compatibility "$_MONGO_MIN_UPDATE_VERSION"
    fi
}

__update_feature_compatibility() {
    mongo --eval "db.adminCommand( { setFeatureCompatibilityVersion: '$1' } )" > /dev/null
}

__install_mongodb() {
    case "$_OS" in
        Ubuntu)
            __add_deb_repo "deb [ arch=$(dpkg --print-architecture) ] http://repo.mongodb.org/apt/ubuntu ${_MONGO_OS_CODENAME}/mongodb-org/$1 multiverse" \
                "mongodb-org-$1" \
                "https://www.mongodb.org/static/pgp/server-$1.asc"
            ;;
        CentOS|RedHatEnterprise|RedHatEnterpriseServer)
            if [ ! -s /etc/yum.repos.d/mongodb-org-$1.repo ]; then
                cat << EOF > /etc/yum.repos.d/mongodb-org-$1.repo
[mongodb-org-$1]
name=MongoDB Repository
baseurl=https://repo.mongodb.org/yum/redhat/\$releasever/mongodb-org/$1/x86_64/
gpgcheck=1
enabled=1
gpgkey=https://www.mongodb.org/static/pgp/server-$1.asc
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
    if [ "$_OS" = "Ubuntu" ]; then
        systemctl enable mongod.service > /dev/null
        systemctl daemon-reload > /dev/null
        systemctl start mongod > /dev/null
        _STOP_MONGO="sudo systemctl stop mongod"
    elif [ "$_OS" = "CentOS" -o "$_OS" = "RedHatEnterprise" -o "$_OS" = "RedHatEnterpriseServer" ]; then
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
    _MONGO_OS_CODENAME="$(lsb_release -cs)"

    if [ "$_OS" != "Ubuntu" -a "$_OS" != "CentOS" -a "$_OS" != "RedHatEnterprise" -a "$_OS" != "RedHatEnterpriseServer" ]; then
        printf "$_ITEM This installer supports Ubuntu, CentOS, and RHEL. \n"
        printf "$_IMPORTANT Your operating system is unsupported."
        exit 1
    fi
}

__gather_pkg_mgr() {
    # _PKG_MGR = 1: APT: Ubuntu 16.04 and Security Onion (Debian)
    # _PKG_MGR = 2: YUM: CentOS (Old RHEL Derivatives)
    # _PKG_MGR = 3: Unsupported
    _PKG_MGR=3
    _PKG_INSTALL=""
    if [ -x /usr/bin/apt-get ];	then
        _PKG_MGR=1
        _PKG_INSTALL="DEBIAN_FRONTEND=noninteractive apt-get -qq install -y"
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

__bro_installed() {
    _BRO_INSTALLED=false
    if __package_installed bro; then
        _BRO_INSTALLED=true
    elif [ -f "/usr/local/bro/bin/bro" ]; then
        _BRO_INSTALLED=true
    fi

    if [ "$_BRO_INSTALLED" = "true" ]; then
        printf "It looks like Bro is installed on this system. This version of RITA uses Zeek.\n"
        printf "For the best results, please stop the script, uninstall Bro, and re-run the script.\n"
        printf "\n"
        printf "Pausing for 20 seconds before continuing.\n"
        _INSTALL_ZEEK=false
        sleep 20
    fi
}


__gather_zeek() {
    _ZEEK_PATH=""
    _ZEEK_PKG_INSTALLED=false
    if __package_installed zeek; then
        _ZEEK_PKG_INSTALLED=true
        _ZEEK_PATH="/opt/zeek/bin"
    fi

    _ZEEK_ONION_INSTALLED=false
    if __package_installed securityonion-bro; then
        _ZEEK_ONION_INSTALLED=true
        _ZEEK_PATH="/opt/zeek/bin"
    fi

    _ZEEK_SOURCE_INSTALLED=false
    if [ -f "/usr/local/zeek/bin/zeek" ]; then
        _ZEEK_SOURCE_INSTALLED=true
        _ZEEK_PATH="/usr/local/zeek/bin"
    fi

    _ZEEK_INSTALLED=false
    if [ $_ZEEK_PKG_INSTALLED = "true" -o $_ZEEK_ONION_INSTALLED = "true" -o $_ZEEK_SOURCE_INSTALLED = "true" ]; then
        _ZEEK_INSTALLED=true
    fi

    if [ "$_INSTALL_ZEEK" = "true" -a "$_ZEEK_INSTALLED" = "true" -a ! -d "$_ZEEK_PATH" ]; then
        printf "$_IMPORTANT An unsupported version of Zeek is installed on this system.\n"
        printf "$_IMPORTANT RITA has not been tested with this version of Zeek and may not function correctly.\n"
        printf "$_IMPORTANT For the best results, please stop this script, uninstall Zeek, and re-run the installer.\n"
        printf "\n"
        printf "$_IMPORTANT Pausing for 20 seconds before continuing. \n"
        _INSTALL_ZEEK=false
        sleep 20
    fi

    _ZEEK_PATH_SCRIPT="/etc/profile.d/zeek-path.sh"
    _ZEEK_PATH_SCRIPT_INSTALLED=false

    if [ -f "$_ZEEK_PATH_SCRIPT" ]; then
        source "$_ZEEK_PATH_SCRIPT"
        _ZEEK_PATH_SCRIPT_INSTALLED=true
    fi

    _ZEEK_IN_PATH=false
    if [ -n "$(type -pf zeek)" ]; then
        _ZEEK_IN_PATH=true
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
    if [ $_ZEEK_INSTALLED = "false" -a $_INSTALL_ZEEK = "true" ]; then
        printf "$_SUBITEM Install Zeek IDS to /opt/zeek \n"
    fi
    if [ $_MONGO_INSTALLED = "false" -a $_INSTALL_MONGO = "true" ]; then
        printf "$_SUBITEM Install MongoDB \n"
    fi
    if ( ! __installation_exist ) && [ $_INSTALL_RITA = "true" ]; then
        printf "$_SUBITEM Install RITA to $_BIN_PATH/rita \n"
        printf "$_SUBITEM Create a runtime directory for RITA in $_VAR_PATH \n"
        printf "$_SUBITEM Create a configuration directory for RITA in $_CONFIG_PATH \n"
    elif [ "$_REINSTALL_RITA" = "true" ]; then
        printf "$_SUBITEM Update RITA at $_BIN_PATH/rita \n"
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
along with necessary dependencies, including Zeek IDS and MongoDB.
Usage:
    ${_NAME} [<arguments>]
Options:
    -h|--help			Show this help message.
    -r|--reinstall			Force reinstalling RITA.
    -v|--version <version>		Specify the version tag of RITA to install instead of master.
    --disable-zeek|--disable-bro	Disable automatic installation of Zeek IDS.
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
