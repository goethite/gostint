#!/bin/bash -e

echo
echo "***************************"
echo "*** Starting BATS Tests ***"
echo "***************************"
echo

vault login root

mongo admin -u gostint_admin -p admin123 --eval "db=db.getSiblingDB('gostint'); db.queues.remove({})"

# Client app role
vault policy write gostint-client - <<EOF
path "auth/token/create" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
path "auth/approle/role/gostint-role/secret-id" {
  capabilities = ["update"]
}
path "transit/encrypt/gostint-role" {
  capabilities = ["update"]
}
EOF

vault write auth/approle/role/gostint-client-role \
  token_ttl=20m \
  token_max_ttl=30m \
  policies="gostint-client"

export APPROLEID=$(vault read auth/approle/role/gostint-client-role/role-id | grep "^role_id" | awk '{print $2;}')
export SECRETID=$(uuid)

vault write auth/approle/role/gostint-client-role/custom-secret-id \
  secret_id=$SECRETID

export TOKEN=$(vault write auth/approle/login \
    role_id=$APPROLEID \
    secret_id=$SECRETID | grep "^token[ \t]" | awk '{print $2;}')

vault login $TOKEN
