#!/bin/sh -xe

# sudo dockerd --pidfile /tmp/docker.pid -H unix:///var/lib/goswim/docker.sock &
(sudo dockerd 2>&1 | grep -v "level=info") &

# drop sudo privs
sudo mv /etc/sudoers /etc/sudoers.DISABLED

sleep 3
echo
export GOSWIM_SSL_CERT="${GOSWIM_SSL_CERT:-/var/lib/goswim/cert.pem}"
export GOSWIM_SSL_KEY="${GOSWIM_SSL_KEY:-/var/lib/goswim/key.pem}"
export GOSWIM_DBURL="${GOSWIM_DBURL:-172.17.0.1:27017}"

/usr/bin/goswim
