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

	printf "[+] Transferring files\n"
	mkdir $_RITADIR
	
	cp -r etc $_RITADIR/etc
	cp -r usr $_RITADIR/usr
	cp LICENSE $_RITADIR/LICENSE

	# This just checks if we're doing a an install during dev (rita would be 
	# the directory from which this is running in that case) or if it's 
	# a part of an installer package with a binary prepacked in bin
	if [ -e bin/rita ] && [ -x bin/rita ] && [ -w /usr/local/bin ]
	then
		printf "[+] Creating symlink to /usr/local/bin/rita\n"
		ln -s $_RITADIR/bin/rita /usr/local/bin/rita
	fi

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
