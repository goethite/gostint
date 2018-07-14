#!/bin/sh -xe

# build goswim docker image and point to vagrant based Vault and MongoDB for
# image testing.  Requires `vagrant up`.  If goswim is also running in the
# vagrant instance, then it may pick up jobs from the queues instead of this
# instance (aka cluster mode).

docker build -t goswim .

# port mapping in Vagrantfile
export VAULT_ADDR="${VAULT_ADDR:-http://172.17.0.1:8300}"

vault login root

token=$(curl -s \
  --request POST \
  --header 'X-Vault-Token: root' \
  --data '{"policies": ["goswim-mongodb-auth"], "ttl": "10m", "num_uses": 2}' \
  ${VAULT_ADDR}/v1/auth/token/create | jq .auth.client_token -r)

roleid=`curl -s --header 'X-Vault-Token: root' \
  ${VAULT_ADDR}/v1/auth/approle/role/goswim-role/role-id | jq .data.role_id -r`

docker run --name goswim -p 3333:3232 \
  --privileged=true \
  -v $(pwd)/etc:/var/lib/goswim \
  -e VAULT_ADDR="$VAULT_ADDR" \
  -e GOSWIM_DBAUTH_TOKEN="$token" \
  -e GOSWIM_ROLEID="$roleid" \
  goswim
