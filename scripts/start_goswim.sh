#!/bin/bash -xe

# Note: DEPRECATED see godo

# Create a one-time token to allow goswim to auth to mongodb on startup
# echo '=== Create onetime token for goswim ====================='
# export ONETIMETOKEN=`curl -s \
#   --request POST \
#   --header 'X-Vault-Token: root' \
#   --data '{"policies": ["goswim-mongodb-auth"], "ttl": "1h", "num_uses": 1}' \
#   ${VAULT_ADDR}/v1/auth/token/create | jq .auth.client_token -r`
#
# echo '=== Starting goswim ====================================='
# go run main.go
