
source $BATS_TEST_DIRNAME/helpers.sh

# These tests only apply to Ubuntu 16.04 Xenial
# [ $OS = "Ubuntu" ] && [ $OS_CODENAME = "xenial" ] || exit 0

@test "rita is installed" {
	_rita_binary="/usr/local/bin/rita"
	_rita_config="/etc/rita/config.yaml"
	_rita_logs="/var/lib/rita/logs"

	ensure_file			$_rita_binary
	ensure_readable 	$_rita_binary
	ensure_executable 	$_rita_binary

	ensure_file			$_rita_config
	ensure_readable		$_rita_config
	ensure_writable		$_rita_config

	ensure_directory	$_rita_logs
	ensure_readable 	$_rita_logs
	ensure_writable		$_rita_logs
	ensure_executable 	$_rita_logs
}

@test "bro is installed" {
	_bro_pkg="/opt/bro"
	_bro_src="/usr/local/bro"

	if [ -d $_bro_pkg ]; then
		_bro_path=$_bro_pkg
	elif [ -d $_bro_src ]; then
		_bro_path=$_bro_src
	else
		echo "bro was not installed"
		exit 1
	fi

	_bro_binary="$_bro_path/bin/bro"
	_broctl_binary="$_bro_path/bin/broctl"
	_bro_node_cfg=""

	ensure_readable		$_bro_binary
	ensure_executable 	$_bro_binary

	ensure_readable		$_broctl_binary
	ensure_executable 	$_broctl_binary
}

@test "bro is configured to start on boot" {
	if [ $(sudo crontab -l | grep 'broctl cron' | wc -l) -eq 0 ]; then
		echo "broctl crontab entry does not exist"
		exit 1
	fi
}

@test "mongo is installed" {
	_mongo_binary="/usr/bin/mongod"

	ensure_file 		$_mongo_binary
	ensure_readable 	$_mongo_binary
	ensure_executable 	$_mongo_binary
}

@test "mongo is configured to start on boot" {
	ensure_exists "/etc/systemd/system/multi-user.target.wants/mongod.service"
}

