#!/bin/bash -e

VAULTVER=0.11.0

# Install and start Vault server in dev mode
wget -qO /tmp/vault.zip https://releases.hashicorp.com/vault/${VAULTVER}/vault_${VAULTVER}_linux_amd64.zip && \
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
sleep 5
vault login root

echo '=== Mocking production mounts of secret engines =========='
vault secrets move secret/ kv/  # v2 at /kv
vault secrets enable -path=secret/ -version=1 kv

echo '=== Enable MongoDB secret engine ========================='
vault secrets enable database

# Configure Vault's MongoDB Secret Engine for our DB instance
# this requires privileged creds for the DB to allow Vault to issue ephemeral
# creds to gostint.  It is recommended to rotate your privileged creds in production.
vault write database/config/gostint-mongodb \
  plugin_name=mongodb-database-plugin \
  allowed_roles="gostint-dbauth-role" \
  connection_url="mongodb://{{username}}:{{password}}@127.0.0.1:27017/admin?ssl=false" \
  username="${MUSER}" \
  password="${MPWD}" && \

vault write database/roles/gostint-dbauth-role \
  db_name=gostint-mongodb \
  creation_statements='{ "db": "gostint", "roles": [{ "role": "readWrite" }] }' \
  default_ttl="10m" \
  max_ttl="24h"

# Create policy to access mongodb secret eng user/pass generator
echo '=== Create policy to access mongodb ====================='
curl -s \
  --request POST \
  --header 'X-Vault-Token: root' \
  --data '{"policy": "path \"database/creds/gostint-dbauth-role\" {\n  capabilities = [\"read\"]\n}"}' \
  ${VAULT_ADDR}/v1/sys/policy/gostint-mongodb-auth

echo '=== Enable transit plugin ==============================='
vault secrets enable transit

echo '=== Create gostint instance transit keyring =============='
vault write -f transit/keys/gostint

# Enable Vault AppRole
echo '=== enable AppRole auth ================================='
vault auth enable approle

# Create policy to access kv secrets for approle
echo '=== Create policy to access secret/ v1 for gostint-role ='
curl -s \
  --request POST \
  --header 'X-Vault-Token: root' \
  --data '{"policy": "path \"secret/*\" {\n  capabilities = [\"read\"]\n}"}' \
  ${VAULT_ADDR}/v1/sys/policy/gostint-approle-secret-v1

echo '=== Create policy to access kv/ v2 for gostint-role =========='
curl -s \
  --request POST \
  --header 'X-Vault-Token: root' \
  --data '{"policy": "path \"kv/*\" {\n  capabilities = [\"read\"]\n}"}' \
  ${VAULT_ADDR}/v1/sys/policy/gostint-approle-kv-v2

# Create policy to access transit decrypt gostint for approle
echo '=== Create policy to access transit decrypt gostint for gostint-role =========='
curl -s \
  --request POST \
  --header 'X-Vault-Token: root' \
  --data '{"policy": "path \"transit/decrypt/gostint\" {\n  capabilities = [\"update\"]\n}"}' \
  ${VAULT_ADDR}/v1/sys/policy/gostint-approle-transit-decrypt-gostint

# Create named role for gostint
echo '=== Create approle role for gostint ======================'
vault write auth/approle/role/gostint-role \
  secret_id_ttl=24h \
  secret_id_num_uses=10000 \
  token_num_uses=10 \
  token_ttl=20m \
  token_max_ttl=30m \
  policies="gostint-approle-secret-v1,gostint-approle-kv-v2,gostint-approle-transit-decrypt-gostint"

# Get RoleID for gostint
export GOSTINT_ROLEID=`vault read -format=yaml -field=data auth/approle/role/gostint-role/role-id | awk '{print $2;}'`
echo "export GOSTINT_ROLEID=$GOSTINT_ROLEID" | tee -a .bashrc
