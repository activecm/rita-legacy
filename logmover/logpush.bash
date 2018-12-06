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
# Note: distributed locking is hard, 
# so we use ssh to lock on the RITA server
LOCK="$REMOTE_LOG_DIR/.rita.read.lock"

# We will store the log data here
DEST_DIR="$REMOTE_LOG_DIR/$COLLECTOR"

# On the RITA server we lock, sleep, and wait
# Note: we grab the sleep pid to kill the shell
# SSH will exit gracefully
SCRIPT='( flock -s 9; sleep infinity & echo $!; wait )9>'"$LOCK"

# We use a named pipe to talk between threaded tasks
LOCK_PIPE=".lock_pipe"

# We want to transfer yesterday's logs
TX_DIR=$LOCAL_LOG_DIR/$(date +%Y-%m-%d)

# Check if yesterday's logs are available
if [ ! -d $TX_DIR ]
then
  echo "No local folder found! Using: $TX_DIR"
  exit 1
fi

# Cleanup the named pipe
trap "rm -f $LOCK_PIPE" EXIT

# Create the named pipe
if [[ ! -p $LOCK_PIPE ]]
then
  mkfifo $LOCK_PIPE
fi

# Run the lock script from the server in the background
# and read output into the channel
# Stream redirection directly into the pipe doesn't work since
# bash will wait until ssh to forward the output
echo $SCRIPT | ssh -i $KEYFILE $USER@$REMOTE "/bin/bash" | {
  read line
  echo $line > $LOCK_PIPE
} &

# Grab the sleep pid
echo "Synchronizing"
lock_pid=$(<$LOCK_PIPE)
echo $lock_pid
echo "Writing"
rsync -a -e "ssh -i $KEYFILE" $TX_DIR $USER@$REMOTE:$DEST_DIR

# kill the sleep process and unlock
ssh -i $KEYFILE $USER@$REMOTE "kill $lock_pid"
