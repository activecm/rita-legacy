#!/bin/sh

# Run this script on the RITA host  after writes
# have started coming into the analysis server
# from the bro nodes. If you start the logpush
# script at 1 am on the nodes, start this script
# at about 1:15 or 2 am. As long as the logpush
# has been run before this script, all will be well. 

# Where RITA is configured to read bro logs from
LOG_DIR=""

##################################################
LOCK_NAME=".rita.read.lock"
LOCK="$LOG_DIR/$LOCK_NAME"

(
  echo "Waiting for exclusive lock"
  flock -x 9
  echo "Gained exclusive lock"
  echo "Parsing"
  find $LOG_DIR -name *.gz -exec gzip -d {} \;
  rita import
  find $LOG_DIR ! -path $LOG_DIR ! -name $LOCK_NAME -exec rm -rf {} +
  echo "Finished parsing"
) 9>$LOCK
echo "Analyzing"
rita analyze
