---
title: Running a Job - the easy way with gostint-client
classes: wide
toc: true
---
## gostint-client
Sister project [gostint-client](https://github.com/goethite/gostint-client) provides
a CLI (and eventually an api library) to simplify securely submitting jobs to
DevOps automation tools running from the gostint API.

The latest release is available from the github [Releases](https://github.com/goethite/gostint-client/releases) page.

### Usage
```
$ ./gostint-client -h
Usage of ./gostint-client:
  -cont-on-warnings
    	Continue to run job even if vault reported warnings when looking up secret refs, overrides value in job-json
  -content string
    	Folder or targz to inject into the container relative to root '/' folder, overrides value in job-json
  -debug
    	Enable debugging
  -entrypoint string
    	JSON array of string parts defining the container's entrypoint, e.g.: '["ansible"]', overrides value in job-json
  -image string
    	Docker image to run job within, overrides value in job-json
  -job-json string
    	JSON Job request
  -poll-interval int
    	Overide default poll interval for results (in seconds) (default 1)
  -qname string
    	Job Queue to submit to, overrides value in job-json
  -run string
    	JSON array of string parts defining the command to run in the container - aka the job, e.g.: '["-m", "ping", "127.0.0.1"]', overrides value in job-json
  -run-dir string
    	Working directory within the container to run the job
  -secret-filetype string
    	Injected secret file type, can be either 'yaml' (default) or 'json', overrides value in job-json (default "yaml")
  -secret-refs string
    	JSON array of strings providing paths to secrets in the Vault to be injected into the job's container, e.g.: '["mysecret@secret/data/my-secret.my-value", ...]', overrides value in job-json
  -url string
    	GoStint API URL, e.g. https://somewhere:3232
  -vault-roleid string
    	Vault App Role ID (can read file e.g. '@role_id.txt')
  -vault-secretid string
    	Vault App Secret ID (can read file e.g. '@secret_id.txt')
  -vault-token string
    	Vault token - used instead of App Role (can read file e.g. '@token.txt')
  -vault-url string
    	Vault API URL, e.g. https://your-vault:8200 - defaults to env var VAULT_ADDR
```

### Debugging with -debug option
```
$ gostint-client -vault-token=@.vault_token \
  -url=https://127.0.0.1:13232 \
  -vault-url=http://127.0.0.1:18200 \
  -image=alpine \
  -run='["cat", "/etc/os-release"]' \
  -debug

2018-08-28T13:11:23+01:00 Validating command line arguments
2018-08-28T13:11:23+01:00 Resolving file argument @.vault_token
2018-08-28T13:11:23+01:00 Building Job Request
2018-08-28T13:11:23+01:00 Getting Vault api connection http://127.0.0.1:18200
2018-08-28T13:11:23+01:00 Authenticating with Vault
2018-08-28T13:11:23+01:00 Getting minimal token to authenticate with GoStint API
2018-08-28T13:11:24+01:00 Getting Wrapped Secret_ID for the AppRole
2018-08-28T13:11:24+01:00 Encrypting the job payload
2018-08-28T13:11:24+01:00 Getting minimal limited use / ttl token for the cubbyhole
2018-08-28T13:11:24+01:00 Putting encrypted payload in a vault cubbyhole
2018-08-28T13:11:24+01:00 Creating job request wrapper to submit
2018-08-28T13:11:24+01:00 Submitting job
2018-08-28T13:11:24+01:00 Response status: 200 OK
...
2018-08-28T13:11:29+01:00 Elapsed time: 5.327 seconds
```

### Run a simple shell command in a container

```
$ gostint-client -vault-token=@.vault_token \
  -url=https://127.0.0.1:13232 \
  -vault-url=http://127.0.0.1:18200 \
  -image=alpine \
  -run='["cat", "/etc/os-release"]'

NAME="Alpine Linux"
ID=alpine
VERSION_ID=3.8.0
PRETTY_NAME="Alpine Linux v3.8"
HOME_URL="http://alpinelinux.org"
BUG_REPORT_URL="http://bugs.alpinelinux.org"
```
### Running Ansible containers
```
$ gostint-client -vault-token=@.vault_token \
  -url=https://127.0.0.1:13232 \
  -vault-url=http://127.0.0.1:18200 \
  -image="jmal98/ansiblecm:2.5.5" \
  -entrypoint='["ansible"]' \
  -run='["--version"]'

ansible 2.5.5
  config file = None
  configured module search path = [u'/tmp/.ansible/plugins/modules', u'/usr/share/ansible/plugins/modules']
  ansible python module location = /usr/lib/python2.7/site-packages/ansible
  executable location = /usr/bin/ansible
  python version = 2.7.14 (default, Dec 14 2017, 15:51:29) [GCC 6.4.0]
```

```
$ gostint-client -vault-token=@.vault_token \
  -url=https://127.0.0.1:13232 \
  -vault-url=http://127.0.0.1:18200 \
  -image="jmal98/ansiblecm:2.5.5" \
  -entrypoint='["ansible"]' \
  -run='["-i", "127.0.0.1 ansible_connection=local,", "-m", "ping", "127.0.0.1"]'

127.0.0.1 | SUCCESS => {
    "changed": false,
    "ping": "pong"
}
```

```
$ gostint-client -vault-token=@.vault_token \
  -url=https://127.0.0.1:13232 \
  -vault-url=http://127.0.0.1:18200 \
  -image="jmal98/ansiblecm:2.5.5" \
  -content=../gostint/tests/content_ansible_play \
  -run='["-i", "hosts", "play1.yml"]'

PLAY [all] *********************************************************************

TASK [Gathering Facts] *********************************************************
ok: [127.0.0.1]

TASK [include_vars] ************************************************************
ok: [127.0.0.1]

TASK [debug] *******************************************************************
ok: [127.0.0.1] => {
    "gostint": {
        "TOKEN": "secret-injected-by-gostint"
    }
}

PLAY RECAP *********************************************************************
127.0.0.1                  : ok=3    changed=0    unreachable=0    failed=0   
```

### Using Vault AppRole Authentication

Create a vault policy for the gostint-client's approle
```
vault policy write gostint-client - <<EOF
path "auth/token/create" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
path "auth/approle/role/gostint-role/secret-id" {
  capabilities = ["update"]
}
path "transit/encrypt/gostint" {
  capabilities = ["update"]
}
EOF
```

Create an AppRole (PUSH mode for this example) for the gostint-client:
```
vault write auth/approle/role/gostint-client-role \
  token_ttl=20m \
  token_max_ttl=30m \
  policies="gostint-client"
```
Get the Role_Id for the AppRole:
```
vault read /auth/approle/role/gostint-client-role/role-id
```
For this example we will use PUSH mode on the AppRole (note the secret_id was a
random uuid) - you would probably prefer to use PULL mode in production:
```
vault write auth/approle/role/gostint-client-role/custom-secret-id \
  secret_id=7a32c590-aacc-11e8-a59c-8b71f9a0c1a4
```

Run gostint-client using the AppRole:
```
$ gostint-client -vault-roleid=43a03f77-7461-d4d2-c14d-76b39ea400d5 \
  -vault-secretid=7a32c590-aacc-11e8-a59c-8b71f9a0c1a4 \
  -url=https://127.0.0.1:13232 \
  -vault-url=http://127.0.0.1:18200 \
  -image=alpine \
  -run='["cat", "/etc/os-release"]'
```
