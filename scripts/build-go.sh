#!/usr/bin/env bash

set -eu
make deps-update
make -C ./hw-event-proxy build-only
