#!/usr/bin/env bash
# gRPC requires GCC 4.9+
set -eu
cd /tmp
curl https://download-ib01.fedoraproject.org/pub/epel/7/aarch64/Packages/a/avr-gcc-4.9.2-1.el7.aarch64.rpm -O
rpm -Uvh avr-gcc-4.9.2-1.el7.aarch64.rpm
yum install avr-gcc
rm avr-gcc-4.9.2-1.el7.aarch64.rpm
