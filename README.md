# goswim - A Shallow RESTful api for Ansible, Terraform ...
... and basically anything you would like to run as jobs in docker containers, authenticated with Hashicorp Vault AppRoles with Secret Injection.

Goal is to be a Highly Available and Scaleable API for automation.

See [Concept Ideas](docs/Concept_Ideas.md)

At this stage this project is a proof-of-concept and under development...

See [build_test_dev script](./build_test_against_dev.sh) for example starting the goswim docker container with the instances of Vault and MongoDb running in the vagrant container.

See [bats tests folder](tests/bats) for example `curl` command based BATs tests, that
demo driving the goswim api to run a selection of Docker container based jobs.
JSON jobs used in these tests are in the respective [tests](tests/) files.

[Dev Notes](docs/devnotes.md)

## Features
* Integrated with Hashicorp Vault AppRole.
* Secrets in Vault can be referenced in a job request, which are then populated
and injected into the job's running container.
* Additional content can be flexibly injected into the job container from the
json request.
* Can run any job in any required docker image, e.g. Ansible, Terraform, busybox,
Powershell, and the versions of the job execution container can be pinned.
* Serialisation queues are dynamic and created on the fly.

## Usage

### Prerequisites
1. A MongoDB service

2. A Hashicorp Vault service
See test setup in [scripts/init_vault.sh](scripts/init_vault.sh) for example of enabling the MongoDB Secret Engine in Vault.

3. SSL Key and Certificate for goswim - `key.pem` and `cert.pem` stored in persistent volume shown below as `/srv/goswim-1/etc`

### Running the goswim docker container
A very basic setup for a single instance of goswim:
```bash
# point to your vault's url
VAULT_ADDR="${VAULT_ADDR:-https://your.vault.host:8200}"

# login to the vault - using your chosen authentication scheme in vault
vault login # to get a <token>

# Request a MongoDB secret engine token for goswim to request an ephemeral
# time-bound username/password pair.
token=$(curl -s \
  --request POST \
  --header 'X-Vault-Token: <token>' \
  --data '{"policies": ["goswim-mongodb-auth"], "ttl": "10m", "num_uses": 2}' \
  ${VAULT_ADDR}/v1/auth/token/create | jq .auth.client_token -r)

# Get goswim's AppRole RoleId from the Vault
roleid=`curl -s --header 'X-Vault-Token: root' \
  ${VAULT_ADDR}/v1/auth/approle/role/goswim-role/role-id | jq .data.role_id -r`

# Run goswim
docker run --init -d \
  --name goswim -p 3232:3232 \
  --privileged=true \
  -v /srv/goswim-1/etc:/var/lib/goswim \
  -e VAULT_ADDR="$VAULT_ADDR" \
  -e GOSWIM_DBAUTH_TOKEN="$token" \
  -e GOSWIM_ROLEID="$roleid" \
  -e GOSWIM_DBURL=your-db-host:27017
  goethite/goswim
```

### TODO: Going HA and Scalable with goswim

## LICENSE - GPLv3

```
Copyright 2018 Graham Lee Bevan <graham.bevan@ntlworld.com>

goswim is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

goswim is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with goswim.  If not, see <https://www.gnu.org/licenses/>.
```
