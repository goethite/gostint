#!/bin/bash -e

# Install and start Vault server in dev mode
wget -qO /tmp/vault.zip https://releases.hashicorp.com/vault/0.10.3/vault_0.10.3_linux_amd64.zip && \
   ( cd /usr/local/bin && unzip /tmp/vault.zip )
rm /tmp/vault.zip
vault -autocomplete-install
echo '=== Starting vault =================================='
(
  cd /tmp
  nohup vault server -dev \
    -dev-root-token-id=root \
    -dev-listen-address="0.0.0.0:8200" \
    >vault.log 2>&1 &
)
echo -e 'export VAULT_ADDR=http://127.0.0.1:8200' >> .bashrc
export VAULT_ADDR=http://127.0.0.1:8200

# Login to vault and configure
echo '=== Logging in to vault =================================='
vault login root

echo '=== Enable MongoDB secret engine ========================='
vault secrets enable database

vault write database/config/goswim-mongodb \
  plugin_name=mongodb-database-plugin \
  allowed_roles="goswim-dbauth-role" \
  connection_url="mongodb://{{username}}:{{password}}@127.0.0.1:27017/admin?ssl=false" \
  username="${MUSER}" \
  password="${MPWD}" && \

vault write database/roles/goswim-dbauth-role \
  db_name=goswim-mongodb \
  creation_statements='{ "db": "goswim", "roles": [{ "role": "readWrite" }] }' \
  default_ttl="10m" \
  max_ttl="24h"

# Enable Vault AppRole
echo '=== enable AppRole auth ================================='
vault auth enable approle

# Create policy to access mongodb secret eng user/pass generator
echo '=== Create policy to access mongodb ====================='
curl -s \
  --request POST \
  --header 'X-Vault-Token: root' \
  --data '{"policy": "path \"database/creds/goswim-dbauth-role\" {\n  capabilities = [\"read\"]\n}"}' \
  ${VAULT_ADDR}/v1/sys/policy/goswim-mongodb-auth
