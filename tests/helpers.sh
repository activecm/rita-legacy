
# e.g. Ubuntu, CentOS
OS="$(lsb_release -is)"
# e.g. trusty, xenial, bionic
OS_CODENAME="$(lsb_release -cs)"


ensure_readable() {
	# Print out any offending path with a readable error msg
	[ -r "$1" ] || echo "$1 is not readable"
	# Perform the actual test to get the error code
	[ -r "$1" ]
}

ensure_writable() {
	# Print out any offending path with a readable error msg
	[ -w "$1" ] || echo "$1 is not writable"
	# Perform the actual test to get the error code
	[ -w "$1" ]
}

ensure_executable() {
	# Print out any offending path with a readable error msg
	[ -x "$1" ] || echo "$1 is not executable"
	# Perform the actual test to get the error code
	[ -x "$1" ]
}

ensure_exists() {
	# Print out any offending path with a readable error msg
	[ -e "$1" ] || echo "$1 does not exist"
	# Perform the actual test to get the error code
	[ -e "$1" ]
}

ensure_file() {
	# Print out any offending path with a readable error msg
	[ -f "$1" ] || echo "$1 is not a file"
	# Perform the actual test to get the error code
	[ -f "$1" ]
}

ensure_directory() {
	# Print out any offending path with a readable error msg
	[ -d "$1" ] || echo "$1 is not a directory"
	# Perform the actual test to get the error code
	[ -d "$1" ]
}