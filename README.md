# gostint - A Shallow RESTful api for Ansible, Terraform ...
... and basically anything you would like to run as jobs in docker containers.
Authenticated and end-to-end encrypted with Hashicorp Vault with Secret Injection.

> gostint:
> : _stint - an allotted amount or piece of work_

Goal is to be a Highly Available and Scaleable Secure API for automation.

See [Concept Ideas](docs/Concept_Ideas.md)

At this stage this project is a proof-of-concept and under development...

Prebuilt releases are available [here](https://github.com/goethite/gostint/releases).

See [build_test_dev script](./build_test_against_dev.sh) for example starting the gostint docker container with the instances of Vault and MongoDb running in the vagrant container.

See [bats tests folder](tests/bats) for example `curl` command based BATs tests, that
demo driving the gostint api to run a selection of Docker container based jobs.
JSON jobs used in these tests are in the respective [tests](tests/) files.

* [Dev Notes](docs/devnotes.md)
* [Job States](docs/jobstates.md)
* [Brainstorming job sequence diagrams](docs/jobsequence.md)

## Features
* Integrated with Hashicorp Vault's AppRole Authentication, Transit end-to-end
  encryption, Cubbyhole, Token Wrapping and KV Secrets.
* Secrets in Vault can be referenced in a job request, which are then injected
  into the job's running container.
* Additional content can be flexibly injected into the job container from the
  json request.
* Can run any job in any required docker image, e.g. Ansible, Terraform, Busybox,
  Powershell, and the versions of the job execution containers can be pinned.
* Serialisation queues are dynamic and created on the fly.

## Usage

### Prerequisites
1. A MongoDB service

2. A Hashicorp Vault service
See test setup in [scripts/init_vault.sh](scripts/init_vault.sh) for example of enabling the MongoDB Secret Engine in Vault.

3. SSL Key and Certificate for gostint - `key.pem` and `cert.pem` stored in persistent volume shown below as `/srv/gostint-1/etc`

### Running the gostint docker container
A very basic setup for a single instance of gostint:
```bash
# point to your vault's url
VAULT_ADDR="${VAULT_ADDR:-https://your.vault.host:8200}"

# login to the vault - using your chosen authentication scheme in vault
vault login # to get a <token>

# Request a MongoDB secret engine token for gostint to request an ephemeral
# time-bound username/password pair.
token=$(curl -s \
  --request POST \
  --header 'X-Vault-Token: <token>' \
  --data '{"policies": ["gostint-mongodb-auth"], "ttl": "10m", "num_uses": 2}' \
  ${VAULT_ADDR}/v1/auth/token/create | jq .auth.client_token -r)

# Get gostint's AppRole RoleId from the Vault
roleid=`curl -s --header 'X-Vault-Token: root' \
  ${VAULT_ADDR}/v1/auth/approle/role/gostint-role/role-id | jq .data.role_id -r`

# Run gostint
docker run --init -d \
  --name gostint -p 3232:3232 \
  --privileged=true \
  -v /srv/gostint-1/etc:/var/lib/gostint \
  -e VAULT_ADDR="$VAULT_ADDR" \
  -e GOSTINT_DBAUTH_TOKEN="$token" \
  -e GOSTINT_ROLEID="$roleid" \
  -e GOSTINT_DBURL=your-db-host:27017
  goethite/gostint
```

### Going HA and Scalable with gostint
See [gostint-helm](https://github.com/goethite/gostint-helm) for (a work-in-progress)
PoC HA deployment of gostint using mongodb, etcd and vault on kubernetes.

### gostint-client
A sister project called [gostint-client](https://github.com/goethite/gostint-client)
is also available to simplify the client side integrations with Hashicorp Vault
and drive the [gostint api](https://goethite.github.io/gostint/docs/1100_api_v1_job/).

## LICENSE - GPLv3

```
Copyright 2018 Graham Lee Bevan <graham.bevan@ntlworld.com>

gostint is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

gostint is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with gostint.  If not, see <https://www.gnu.org/licenses/>.
```
