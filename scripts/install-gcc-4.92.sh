#!/usr/bin/env bash
# gRPC requires GCC 4.9+
set -eu
yum install -y libmpc-devel mpfr-devel gmp-devel bzip2 gcc-c++ make
cd /tmp
curl https://ftp.gnu.org/gnu/gcc/gcc-4.9.2/gcc-4.9.2.tar.bz2 -O
tar xvfj gcc-4.9.2.tar.bz2
cd gcc-4.9.2
./configure --disable-multilib --enable-languages=c,c++
make -j 4
make install
rm -f /tmp/gcc-4.9.2.tar.bz2
rm -rf /tmp/gcc-4.9.2