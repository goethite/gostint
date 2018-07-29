---
title: Getting Started with GoStint
classes: wide
---
## Getting started with a Vagrant Development Environment
* Prereqs:
  * Docker
  * Vagrant
* Clone this project.
* Start up the vagrant docker instance:
  ```
  gostint $ vagrant up
  ```
* SSH into the Vagrant instance:
  ```
  gostint $ vagrant ssh
  vagrant@23e208f27e53:~$
  ```
* Change to the gostint src folder mapped into the container:
  ```
  vagrant@23e208f27e53:~$ cd go/src/github.com/gbevan/gostint
  vagrant@23e208f27e53:~/go/src/github.com/gbevan/gostint$
  ```
* Start gostint using `godo`:
  ```
  vagrant@23e208f27e53:~/go/src/github.com/gbevan/gostint$ godo
  ```
  (optional parameter `--watch` will restart if changes are detected)

  You should see a log similar to:
  ```
  ...
  2018/07/29 15:37:39 Starting gostint service
  2018/07/29 15:37:39 Dialing Mongodb
  2018/07/29 15:37:39 Logging in to gostint db
  ```
  The gostint server is now running with it's api accessible at
  https://127.0.0.1:3232

* In another vagrant ssh window, you can run the BATs api tests:
  ```
  gostint $ vagrant ssh
  vagrant@23e208f27e53:~$ cd go/src/github.com/gbevan/gostint
  vagrant@23e208f27e53:~$ godo test
  ...
  ```
The Vagrant container starts up the dev instance of MongodDB and Vault automatically.

## Simple Production Deployment
Note: this is a work-in-progress...

This example is to demo a very simply prod deployment.  Ideally in production
each of the components would be configured for High Availability and Scalability.

![Simple Prod](../diagrams/prod_simple.png)

* Start a MongoDB docker service - see the official
  [mongo](https://hub.docker.com/_/mongo/) image.
  Configure:
  ```
  mongo admin --eval "db.createUser({user: 'your_db_user', pwd: 'your_db_password', roles: [{role: 'root', db: 'admin'}]})"
  ```
* Start a Hashicorp Vault service - if deploying to a real production
  environment then consider their [reference architecture](https://www.vaultproject.io/guides/operations/reference-architecture.html)
  guide, esp around hardening recommendations - for now you can use the official
  [vault](https://hub.docker.com/_/vault/) docker image.
  Configure vault for gostint:
  ```
  export VAULT_ADDR="https://your_vault_host:8200"
  vault login your_root_token
  vault secrets enable database

  # Configure Vault's MongoDB Secret Engine for our DB instance
  # this requires privileged creds for the DB to allow Vault to issue ephemeral
  # creds to gostint.  It is recommended to rotate your privileged creds in production.
  vault write database/config/gostint-mongodb \
    plugin_name=mongodb-database-plugin \
    allowed_roles="gostint-dbauth-role" \
    connection_url="mongodb://{{username}}:{{password}}@your_db_host:27017/admin?ssl=false" \
    username="your_db_user" \
    password="your_db_password"

  vault write database/roles/gostint-dbauth-role \
    db_name=gostint-mongodb \
    creation_statements='{ "db": "gostint", "roles": [{ "role": "readWrite" }] }' \
    default_ttl="10m" \
    max_ttl="24h"

  # Create policy to access mongodb secret eng user/pass generator
  curl -s \
    --request POST \
    --header 'X-Vault-Token: your_root_token' \
    --data '{"policy": "path \"database/creds/gostint-dbauth-role\" {\n  capabilities = [\"read\"]\n}"}' \
    ${VAULT_ADDR}/v1/sys/policy/gostint-mongodb-auth

  vault secrets enable transit
  vault write -f transit/keys/gostint

  # Enable Vault AppRole
  vault auth enable approle

  # Create policy to access kv secrets for approle
  curl -s \
    --request POST \
    --header 'X-Vault-Token: your_root_token' \
    --data '{"policy": "path \"secret/*\" {\n  capabilities = [\"read\"]\n}"}' \
    ${VAULT_ADDR}/v1/sys/policy/gostint-approle-kv

  # Create policy to access transit decrypt gostint for approle
  curl -s \
    --request POST \
    --header 'X-Vault-Token: your_root_token' \
    --data '{"policy": "path \"transit/decrypt/gostint\" {\n  capabilities = [\"update\"]\n}"}' \
    ${VAULT_ADDR}/v1/sys/policy/gostint-approle-transit-decrypt-gostint

  # Create named role for gostint
  vault write auth/approle/role/gostint-role \
    secret_id_ttl=24h \
    secret_id_num_uses=10000 \
    token_num_uses=10 \
    token_ttl=20m \
    token_max_ttl=30m \
    policies="gostint-approle-kv,gostint-approle-transit-decrypt-gostint"

  ```
* Start gostint:
  This will need a `deployment_token_from_vault` issued from a privileged persona
  to allow this deployment to setup the required policies etc (dont forget to revoke
  tokens once you are finished with them).
  ```
  export VAULT_ADDR="https://your_vault_host:8200"

  # Request a MongoDB secret engine token for gostint to request an ephemeral
  # time-bound username/password pair.
  token=$(curl -s \
    --request POST \
    --header 'X-Vault-Token: deployment_token_from_vault' \
    --data '{"policies": ["gostint-mongodb-auth"], "ttl": "10m", "num_uses": 2}' \
    ${VAULT_ADDR}/v1/auth/token/create | jq .auth.client_token -r)

  # Get gostint's AppRole RoleId from the Vault
  roleid=`curl -s --header 'X-Vault-Token: deployment_token_from_vault' \
    ${VAULT_ADDR}/v1/auth/approle/role/gostint-role/role-id | jq .data.role_id -r`

  # Run gostint
  docker run --init -d \
  --name gostint -p 3232:3232 \
  --privileged=true \
  -v your_gostint_cfg/etc:/var/lib/gostint \
  --volume /etc/localtime:/etc/localtime:ro \
  --volume /etc/timezone:/etc/timezone:ro \
  -e VAULT_ADDR="$VAULT_ADDR" \
  -e GOSTINT_DBAUTH_TOKEN="$token" \
  -e GOSTINT_ROLEID="$roleid" \
  goethite/gostint:v?.?.?
  ```
  Note: `--privileged=true` is currently required to allow docker-in-docker.

  `your_gostint_cfg/etc` needs to contain your gostint instance's TLS private
  key and certificate, named `key.pem`and `cert.pem` respectively.
