#!/usr/bin/env bash

set -eu
export GOFLAGS=
go mod init tmp
make -C ./hw-event-proxy build-only
