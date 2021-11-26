#!/bin/bash

# Always exit on errors.
set -e

# Trap sigterm
function exitonsigterm() {
  echo "Trapped sigterm, exiting."
  exit 0
}
trap exitonsigterm SIGTERM

/redfish-event-test
status=$?
if [ $status -ne 0 ]; then
  echo "Failed to start redfish-event-test: $status"
  exit $status
fi
