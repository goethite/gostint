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

# Configure Vault's MongoDB Secret Engine for our DB instance
# this requires privileged creds for the DB to allow Vault to issue ephemeral
# creds to goswim.  It is recommended to rotate your privileged creds in production.
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

# Create policy to access mongodb secret eng user/pass generator
echo '=== Create policy to access mongodb ====================='
curl -s \
  --request POST \
  --header 'X-Vault-Token: root' \
  --data '{"policy": "path \"database/creds/goswim-dbauth-role\" {\n  capabilities = [\"read\"]\n}"}' \
  ${VAULT_ADDR}/v1/sys/policy/goswim-mongodb-auth

echo '=== Enable transit plugin ==============================='
vault secrets enable transit

echo '=== Create goswim instance transit keyring =============='
vault write -f transit/keys/goswim

# Enable Vault AppRole
echo '=== enable AppRole auth ================================='
vault auth enable approle

# Create policy to access kv secrets for approle
echo '=== Create policy to access kv for goswim-role =========='
curl -s \
  --request POST \
  --header 'X-Vault-Token: root' \
  --data '{"policy": "path \"secret/*\" {\n  capabilities = [\"read\"]\n}"}' \
  ${VAULT_ADDR}/v1/sys/policy/goswim-approle-kv

# Create policy to access transit decrypt goswim for approle
echo '=== Create policy to access transit decrypt goswim for goswim-role =========='
curl -s \
  --request POST \
  --header 'X-Vault-Token: root' \
  --data '{"policy": "path \"transit/decrypt/goswim\" {\n  capabilities = [\"update\"]\n}"}' \
  ${VAULT_ADDR}/v1/sys/policy/goswim-approle-transit-decrypt-goswim

# Create named role for goswim
echo '=== Create approle role for goswim ======================'
vault write auth/approle/role/goswim-role \
  secret_id_ttl=24h \
  secret_id_num_uses=10000 \
  token_num_uses=10 \
  token_ttl=20m \
  token_max_ttl=30m \
  policies="goswim-approle-kv,goswim-approle-transit-decrypt-goswim"

# Get RoleID for goswim
export GOSWIM_ROLEID=`vault read -format=yaml -field=data auth/approle/role/goswim-role/role-id | awk '{print $2;}'`
echo "export GOSWIM_ROLEID=$GOSWIM_ROLEID" | tee -a .bashrc
