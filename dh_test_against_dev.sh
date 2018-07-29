#!/bin/sh -xe
#
# Usage:
#   ./dh_test_against_dev.sh tag
if [ "$1" = "" ]
then
  echo "ERROR: tag parameter required" >&2
  exit 1
fi

TAG="$1"

# point to vagrant based Vault and MongoDB for image testing.
# Requires `vagrant up`.  If gostint is also running in the
# vagrant instance, then it may pick up jobs from the queues instead of this
# instance (aka cluster mode).

# docker build -t gostint .

# port mapping in Vagrantfile
export VAULT_ADDR="${VAULT_ADDR:-http://172.17.0.1:8300}"

# login to the vault
# vault login root

# Request a MongoDB secret engine token for gostint to request an ephemeral
# time-bound username/password pair.
token=$(curl -s \
  --request POST \
  --header 'X-Vault-Token: root' \
  --data '{"policies": ["gostint-mongodb-auth"], "ttl": "10m", "num_uses": 2}' \
  ${VAULT_ADDR}/v1/auth/token/create | jq .auth.client_token -r)

# Get gostint's AppRole RoleId from the Vault
roleid=`curl -s --header 'X-Vault-Token: root' \
  ${VAULT_ADDR}/v1/auth/approle/role/gostint-role/role-id | jq .data.role_id -r`

# Cleanup any previous runs in Dev
docker stop goethite/gostint:$TAG || /bin/true
docker rm goethite/gostint:$TAG || /bin/true

# Run gostint in foreground to allow monitoring of the log output in the
# terminal.
docker run --init -t \
  --name gostint -p 3433:3232 \
  --privileged=true \
  -v $(pwd)/etc:/var/lib/gostint \
  --volume /etc/localtime:/etc/localtime:ro \
  --volume /etc/timezone:/etc/timezone:ro \
  -e VAULT_ADDR="$VAULT_ADDR" \
  -e GOSTINT_DBAUTH_TOKEN="$token" \
  -e GOSTINT_ROLEID="$roleid" \
  goethite/gostint:$TAG
