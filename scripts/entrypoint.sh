#!/bin/bash

# Always exit on errors.
set -e

# Trap sigterm
function exitonsigterm() {
  echo "Trapped sigterm, exiting."
  exit 0
}
trap exitonsigterm SIGTERM

# Start the message-parser
python3 /message-parser/server.py &
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start message-parser: $status"
  exit $status
fi

# Start the hw-event-proxy
/hw-event-proxy
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start hw-event-proxy: $status"
  exit $status
fi
