#!/bin/bash

#Version 0.3.1

#This sends any bro logs less than three days old to the rita/aihunter server.  It only sends logs of these types:
#conn., dns., http., ssl., x509., and known_certs.  Any logs that already exist on the target system are not retransferred.

#Before using this, run these on the rita/aihunter server:
#sudo adduser dataimport
#sudo passwd dataimport
#sudo mkdir -p /opt/bro/remotelogs/ /home/dataimport/.ssh/
#add the dataimport user's ssh public key to /home/dataimport/.ssh/authorized_keys in the rita/aihunter server
#sudo chown -R dataimport /opt/bro/remotelogs/ /home/dataimport/.ssh/
#sudo chmod go-rwx -R /home/dataimport/.ssh/


default_user_on_aihunter='dataimport'


can_ssh () {
	#Test that we can reach the target system over ssh.
	success_code=1
	if [ "$1" = "127.0.0.1" ]; then
		success_code=0
	elif [ -n "$1" ]; then
		token="$RANDOM.$RANDOM"
		if [ "$2" = "-o" -a "$3" = 'PasswordAuthentication=no' ]; then
			status "Attempting to verify that we can ssh to $1"
		else
			status "Attempting to verify that we can ssh to $@ - you may need to provide a password to access this system."
		fi
		ssh_out=`ssh "$@" '/bin/echo '"$token"`
		if [ "$token" = "$ssh_out" ]; then
			#status "successfully connected to $@"
			success_code=0
		#else
			#status "cannot connect to $@"
		fi
	else
		fail "Please supply an ssh target as a command line parameter to can_ssh"
	fi

	return $success_code
}


fail () {
	echo "$@ , exiting." >&2
	exit 1
}


status () {
	echo "==== $@"
}


usage () {
	echo 'Usage: '"$0"' [--localdir /local/top/dir/] [--dest where_to_ssh] [--remotedir /remote/top/dir/] [--rsyncparams '"' --aparam --anotherparam '"']' >&2
	echo 'The optional --dest can be a hostname, IP, user@hostname, user@ip, or any label in an ~/.ssh/config stanza' >&2
	echo 'If left off, we use the "Location" field from /etc/rita/agent.yaml' >&2
	echo 'The user@... format is discouraged - we want to use dataimport@... on the remote server.' >&2
	echo '' >&2
	echo 'The optional --localdir is where the Bro/Zeek logs can be found on this system system.' >&2
	echo 'If you look in this directory, it should contain at least a directory or symlink called current .' >&2
	echo 'By default we will look in common locations for this directory tree.' >&2
	echo '' >&2
	echo 'The optional --remotedir is where you want the Bro/Zeek logs to end up on the target system.' >&2
	echo 'If left off, it will be /opt/bro/remotelogs/$my_id/' >&2
	echo '' >&2
	echo 'The optional --rsyncparams allows you to specify parameters for rsync.  MAKE SURE to enclose the entire block in a pair of single quotes.  Suggestions:' >&2
	echo '	--bwlimit=NNN	#Limit bandwidth used to NNN kilobytes/sec' >&2
	echo '	-v		#Verbose; list out the files being transferred, discouraged if running from cron' >&2
	echo '	-q		#Turn off any messages that are not errors, encouraged if running from cron' >&2
	echo '	--dry-run	#Test, do not actually transfer files' >&2
	echo '	DO NOT add --compress ; the files we are sending are already compressed.' >&2
	exit
}


require_util () {
	#Returns true if all binaries listed as parameters exist somewhere in the path, False if one or more missing.
        while [ -n "$1" ]; do
                if ! type -path "$1" >/dev/null 2>/dev/null ; then
                        echo Missing utility "$1". Please install it. >&2
                        return 1        #False, app is not available.
                fi
                shift
        done
        return 0        #True, app is there.
} #End of requireutil


#Check that we have basic tools to continue
require_util awk date egrep find grep hostname ip nice rsync sed ssh sort tr		|| fail "Missing a required utility"

#ionice is not stricly required; if it exists we'll use it to give all other processes on the system first access to the disk, effectively eliminating the chance that we cause dropped packets from disk contention.
if type -path ionice >/dev/null 2>/dev/null ; then
	nice_me=' ionice -c 3 nice -n 19 '
else
	nice_me=' nice -n 19 '
fi

while [ -n "$1" ]; do
	if [ "z$1" = "z--localdir" -a -e "$2" ]; then
		local_tld="$2"
		shift
	elif [ "z$1" = "z--remotedir" -a -n "$2" ]; then
		remote_top_dir="$2"
		shift
	elif [ "z$1" = "z--dest" -a -n "$2" ]; then
		if echo "$2" | grep -q '@' ; then
			#User has supplied an "@" symbol in target system, do not add $default_user_on_aihunter
			aih_location="${2}"
		else
			#No "@" symbol in target system, force username to $default_user_on_aihunter
			aih_location="${default_user_on_aihunter}@${2}"
		fi

		shift
	elif [ "z$1" = "z--rsyncparams" -a -n "$2" ]; then
		rsyncparams="$2"
		shift
	elif [ "z$1" = "z-h" -o "z$1" = "z--help" ]; then
		usage
	else
		usage
	fi

	shift
done


if [ -z "$rsyncparams" ]; then
	rsyncparams=" -q "
fi

#Where should we send the bro logs?
if [ -z "$aih_location" ]; then
	if [ -s /etc/rita/agent.yaml ]; then
		aih_location="${default_user_on_aihunter}@`grep '^[^#]*DatabaseLocation' /etc/rita/agent.yaml | sed -e 's/^.*DatabaseLocation:*\W*//'`"
	else
		fail "Destination not set on the command line and no /etc/rita/agent.yaml file to autodetect destination."
	fi
fi

#Find a unique name for this bro node
if [ -s /etc/rita/agent.yaml -a -n "`grep '^[^#]*Name' /etc/rita/agent.yaml | sed -e 's/^.*Name:*\W*//'`" ]; then
	#Manually setting the hostname to use in agent.yaml is preferred...
	my_id=`grep '^[^#]*Name' /etc/rita/agent.yaml | sed -e 's/^.*Name:*\W*//'`
else
	#...but if no name is forced, we use the short hostname + the primary IP, which should be unique.
	#Following is short form of the hostname, then "__", then the primary IP ipv4 address (one for the default route) of the system.
	#The tr command strips off spaces or odd characters in hostname
	my_id=`hostname -s | tr -dc 'a-zA-Z0-9_.:-'`"__"`ip route get 8.8.8.8 | awk '{print $NF;exit}'`
fi

if [ -z "$remote_top_dir" ]; then
	remote_top_dir="/opt/bro/remotelogs/$my_id/"
fi

#Make sure we can ssh to the aihunter system first
if ! can_ssh "$aih_location" "-o" 'PasswordAuthentication=no' ; then
	if [ -s ~/.ssh/id_rsa -a -s ~/.ssh/id_rsa.pub ]; then
		status "Transferring the RSA key to $aih_location - please provide the password when prompted"
		cat ~/.ssh/{id_dsa.pub,id_ecdsa.pub,id_rsa.pub} 2>/dev/null \
		 | ssh "$aih_location" 'mkdir -p .ssh ; cat >>.ssh/authorized_keys ; chmod go-rwx ./ .ssh/ .ssh/authorized_keys'
	elif [ -s ~/.ssh/id_rsa -o -s ~/.ssh/id_rsa.pub ]; then
		fail "Unable to ssh to $aih_location, and one of the keys exist.  Please transfer the public key to $aih_location, make sure you can ssh from here, and rerun this script"
	elif [ ! type -path ssh-keygen >/dev/null 2>/dev/null ]; then
		fail "Unable to ssh to $aih_location, and we do not have a key generator.  Please create a keypair, transfer the public key to $aih_location, make sure you can ssh from here, and rerun this script"
	else
		#Create ssh key if it doesn't exist, and push to aihunter server or ask user to do so.
		status "Creating a new RSA key with no passphrase"
		ssh-keygen -b 2048 -t rsa -N '' -f ~/.ssh/id_rsa
		status "Transferring the RSA key to $aih_location - please provide the password when prompted"
		cat ~/.ssh/{id_dsa.pub,id_ecdsa.pub,id_rsa.pub} 2>/dev/null \
		 | ssh "$aih_location" 'mkdir -p .ssh ; cat >>.ssh/authorized_keys ; chmod go-rwx ./ .ssh/ .ssh/authorized_keys'
	fi

	if ! can_ssh "$aih_location" "-o" 'PasswordAuthentication=no' ; then
		fail "Unable to ssh to $aih_location using something other than a password"
	fi
fi

#What local directory holds the bro logs?  If we add any that do _not_ end in /logs/ , we need to adjust the sed command in send_candidates= below.
#Make sure the directory ends in a "/".
if [ -z "$local_tld" ]; then
	if [ -d /storage/bro/logs/ ]; then				#Custom
		local_tld='/storage/bro/logs/'
	elif [ -d /opt/bro/logs/ ]; then				#Bro as installed by Rita
		local_tld='/opt/bro/logs/'
	elif [ -d /usr/local/bro/logs/ ]; then				#Bro default
		local_tld='/usr/local/bro/logs/'
	elif [ -d /var/lib/docker/volumes/var_log_bro/_data/ ]; then	#Blue vector
		local_tld='/var/lib/docker/volumes/var_log_bro/_data/'
	elif [ -d /nsm/bro/logs/ ]; then				#Security onion
		local_tld='/nsm/bro/logs/'
	else
		fail 'Unable to locate top level directory for bro logs, please rerun script, specifying the top level path to bro logs with --localdir .'
	fi
fi

today=`date '+%Y-%m-%d'`
yesterday=`date '+%Y-%m-%d' --date=yesterday`
twoda=`date '+%Y-%m-%d' --date='2 days ago'`
threeda=`date '+%Y-%m-%d' --date='3 days ago'`

status "Sending logs to rita/aihunter server $aih_location , My name: $my_id , local dir: $local_tld , remote dir: $remote_top_dir"

status "Preparing remote directories"
ssh "$aih_location" "mkdir -p ${remote_top_dir}/$today/ ${remote_top_dir}/$yesterday/ ${remote_top_dir}/$twoda/ ${remote_top_dir}/$threeda/ ${remote_top_dir}/current/"


send_candidates=`find "$local_tld" -type f -mtime -3 -iname '*.gz' | egrep '(conn\.|dns\.|http\.|ssl\.|x509\.|known_certs\.)' | sed -e 's@^.*/logs/@@' -e 's@^.*/_data/@@' | sort -u`
cd "$local_tld" || fail "Unable to change to $local_tld"
status "Transferring files to $aih_location"
$nice_me rsync $rsyncparams -avR -e ssh $send_candidates "$aih_location:${remote_top_dir}/" --delay-updates

#Note: after we added a user option to set the destination dir, we remove the --temp-dir option as this dir may not be on the same mount point as the destination dir.
#rsync will put temporary files in a .~tmp~ directory under each destination subdir.
#Originally:  --temp-dir="/opt/bro/tmp/$my_id/"

