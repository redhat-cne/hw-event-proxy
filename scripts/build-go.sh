#!/usr/bin/env bash

set -eu
# Export GO111MODULE=on to enable project to be built from within GOPATH/src
export GO111MODULE=on
export GOFLAGS=
make -C ./hw-event-proxy build-only
