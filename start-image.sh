#!/bin/sh -xe

# sudo dockerd --pidfile /tmp/docker.pid -H unix:///var/lib/gostint/docker.sock &
(sudo dockerd -s vfs 2>&1 | grep -v "level=info") &

# drop sudo privs
sudo mv /etc/sudoers /etc/sudoers.DISABLED

sleep 3
echo
export GOSTINT_SSL_CERT="${GOSTINT_SSL_CERT:-/var/lib/gostint/cert.pem}"
export GOSTINT_SSL_KEY="${GOSTINT_SSL_KEY:-/var/lib/gostint/key.pem}"
export GOSTINT_DBURL="${GOSTINT_DBURL:-172.17.0.1:27017}"

/usr/bin/gostint
