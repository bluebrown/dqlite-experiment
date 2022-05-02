#!/usr/bin/env sh

set -e

# install build tools
#
apt-get -y update
apt-get -y --no-install-recommends install autoconf make

# get & compile raft
# https://github.com/canonical/raft
#
git clone https://github.com/canonical/raft.git /tmp/raft
cd /tmp/raft
apt-get install -y libtool liblz4-dev libuv1-dev
autoreconf -i
./configure
make
make install
cd -
rm -rf /tmp/raft

# get & compile dqlite
# https://github.com/canonical/dqlite
#
git clone https://github.com/canonical/dqlite.git /tmp/dqlite
cd /tmp/dqlite
apt-get install -y libsqlite3-dev
autoreconf -i
./configure
make
make install
cd -
rm -rf /tmp/dqlite

# get & compile sqlite3
# https://www.sqlite.org/download.html
#
cd /tmp
curl -fsSLO https://www.sqlite.org/2022/sqlite-autoconf-3380300.tar.gz
tar -xzf sqlite-autoconf-3380300.tar.gz
cd sqlite-autoconf-3380300
autoreconf -i
./configure
make
make install
cd -
rm -rf /tmp/sqlite-autoconf-3380300.tar.gz /tmp/sqlite-autoconf-3380300

# configure dynamic linker run-time bindings
#
ldconfig
