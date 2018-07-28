#!/bin/bash -e

apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv 2930ADAE8CAF5059EE73BB4B58712A2291FA4AD5
echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu xenial/mongodb-org/3.6 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-3.6.list

apt update
apt-get install -y mongodb-org

mongod --config /etc/mongod.conf --fork --smallfiles --auth --bind_ip 0.0.0.0

# Wait for MongoDB to become available and config gostint root/admin user
(
  echo "=== Waiting for MongoDB"
  until mongo --host=127.0.0.1:27017 --eval 'print("waited for connection")'; do
    sleep 60
  done
  echo "=== Passed MongoDB"
)
echo '=== Set mongodb root admin pw for dev'
MUSER='gostint_admin'
MPWD='admin123'
mongo admin --eval "db.createUser({user: '${MUSER}', pwd: '${MPWD}', roles: [{role: 'root', db: 'admin'}]})"
