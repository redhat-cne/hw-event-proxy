#!/usr/bin/env bash

set -eu
make -C ./hw-event-proxy deps-update
make -C ./hw-event-proxy build-only
