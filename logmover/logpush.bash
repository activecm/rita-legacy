#!/bin/bash

# This script pushes logs to the RITA server
# Put this in an unprivileged user's cron tab on
# your bro nodes at some time around 1 am,
# ideally some time after bro has safely
# archived yesterday's logs


# The user to connect to the RITA server with
USER=""

# The address of the RITA server
REMOTE=""

# The location of the RITA logs directory
REMOTE_LOG_DIR=""

# The location of the local bro logs
LOCAL_LOG_DIR=""

# The name of this collector node
# The logs will be stored at REMOTE_LOG_DIR/COLLECTOR
COLLECTOR=""

# The ssh key to connect to the RITA server with
KEYFILE=""

##################################################
# We use shared locks for writing to the server,
# and an exclusive lock for parsing
# Each lock is backed by this file.
# NOTE: $REMOTE_LOG_DIR may not be on NFS.
LOCK="$REMOTE_LOG_DIR/.logpush.lock"

# We will store the log data here
DEST_DIR="$REMOTE_LOG_DIR/$COLLECTOR"

# We want to transfer yesterday's logs
TX_DIR=$LOCAL_LOG_DIR/$(date -d "yesterday" +%Y-%m-%d)

# The SERVER_SCRIPT is used to obtain a shared lock
# on the receiving server before starting an rsync
# daemon to receive data. The lock is held
# until the script exits. The script will exit
# once rsync has finished.
read -r -d '' SERVER_SCRIPT << EOF
main() {
  {
    flock -s 9;
    rsync \$@
    local exit_code=\$?
    return \${exit_code}
  } 9>$LOCK
}

main
EOF

# Check if yesterday's logs are available
if [ ! -d $TX_DIR ]
then
  echo "No local folder found! Searched for: $TX_DIR"
  exit 1
fi

# Send the logs
echo "Sending $TX_DIR to $USER@$REMOTE:$DEST_DIR"
rsync -a -e "ssh -i $KEYFILE" --rsync-path="$SERVER_SCRIPT" $TX_DIR $USER@$REMOTE:$DEST_DIR
