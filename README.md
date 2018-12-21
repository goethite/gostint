# gostint - A Shallow RESTful api for Ansible, Terraform ...
... and basically anything you would like to run as jobs in docker containers.
Authenticated and end-to-end encrypted with Hashicorp Vault with Secret Injection
* https://goethite.github.io/gostint/

> gostint:
> : _stint - an allotted amount or piece of work_

Goal is to be a Highly Available and Scaleable Secure API for automation.

See [Concept Ideas](docs/Concept_Ideas.md)

At this stage this project is a MVP and under development / review...

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

### Enabling the gostint UI
To enable the experimental web UI in gostint, simply pass it `GOSTINT_UI=1`:
```bash
... GOSTINT_UI=1 gostint
```
Access the UI at https://127.0.0.1:3232

## Developer Guide

Development and testing is done in a Vagrant/Docker environment:
```bash
$ vagrant up
...
$ vagrant ssh
```
The environment should already be running an instance of MongoDB and Hashicorp Vault:
```bash
vagrant@2c6839c78fbd:~$ ps -ef
UID         PID   PPID  C STIME TTY          TIME CMD
root          1      0  0 11:09 ?        00:00:00 /usr/sbin/sshd -D -e
root       2498      1  0 11:10 ?        00:00:02 dockerd -s vfs
root       2526   2498  1 11:10 ?        00:00:04 containerd --config /var/run/docker/containerd/containerd.toml --log-level info
root       3256      1  1 11:10 ?        00:00:05 mongod --config /etc/mongod.conf --fork --smallfiles --auth --bind_ip 0.0.0.0
root       3309      1  0 11:10 ?        00:00:02 vault server -dev -dev-root-token-id=root -dev-listen-address=0.0.0.0:8200
root       3608      1  0 11:15 ?        00:00:00 sshd: vagrant [priv]
vagrant    3610   3608  0 11:15 ?        00:00:00 sshd: vagrant@pts/0
vagrant    3611   3610  0 11:15 pts/0    00:00:00 -bash
vagrant    3629   3611  0 11:17 pts/0    00:00:00 ps -ef
```
Notice it is also running an instance of Docker-in-Docker (the vagrant instance
runs the docker container in `privileged` mode to support this).

Change to the gostint source folder (mapped by vagrant from your gostint git
clone folder):
```bash
vagrant@2c6839c78fbd:~$ cd go/src/github.com/gbevan/gostint
```

and run `godo` to build and start the gostint application:
```bash
~/go/src/github.com/gbevan/gostint$ godo
Success! You are now authenticated. The token information displayed below
is already stored in the token helper. You do NOT need to run "vault login"
again. Future Vault requests will automatically use this token.

Key                  Value
---                  -----
token                root
token_accessor       4mOso4ZzgZ9PR5wjDwhM2YiK
token_duration       ∞
token_renewable      false
token_policies       ["root"]
identity_policies    []
policies             ["root"]
Success! Data written to: secret/my-secret
Success! Data written to: secret/my-form
Key              Value
---              -----
created_time     2018-11-24T11:24:21.632236029Z
deletion_time    n/a
destroyed        false
version          3
Key              Value
---              -----
created_time     2018-11-24T11:24:21.665699192Z
deletion_time    n/a
destroyed        false
version          3
gettoken>mocksecrets 187ms
7lbKwKMayAkJnwUmHmnmLgge
default>gettoken 12ms
default 317ms

                              |   _)         |
           _` |   _ \    __|  __|  |  __ \   __|
          (   |  (   | \__ \  |    |  |   |  |
         \__, | \___/  ____/ \__| _| _|  _| \__|
         |___/

Copyright 2018 Graham Lee Bevan <graham.bevan@ntlworld.com>
               Licensed under the GNU GPLv3

https://goethite.github.io/gostint/
https://github.com/goethite

2018/11/24 11:24:22 INFO: Starting gostint...
2018/11/24 11:24:22 INFO: gostint listening on https port 3232
```

`godo` can also run in `watch` mode, so it automatically restarts when you make
changes to the code:
```bash
~/go/src/github.com/gbevan/gostint$ godo --watch
```

To run the BATS test suite (in another terminal session):
```bash
~/go/src/github.com/gbevan/gostint$ godo test

***************************
*** Starting BATS Tests ***
***************************
...
```

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
