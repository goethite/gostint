---
title: API v1/api/job
classes: wide
#toc: true
---

|v1/api/job | Description                          |
|-------|--------------------------------------|
| POST / | Post a new job to the queue         |
| POST /kill/{id} | Request job {id} be killed |
| GET /{id} | Get the job by {id}              |
| DELETE /{id} | Delete the job by {id}        |

Paramters that can be specified in a POSTed job request:

| Parameter | |
|-----------|-|
| qname     | Name of the serialization queue to submit to |
| cubby_token | One-time token from Vault to retrieve a payload its cubbyhole |
| cubby_path | Path, in Vault, to the cubbyhole on the above token |
| wrap_secret_id | Wrapping Token for the GoStint's AppRole SecretID, from Vault.  This allows GoStint to complete its authentication to the Vault |

The job payload encrypted and placed in the cubbyhole can contain:

| Parameter | |
|-----------|-|
| container_image | Image name in DockerHub to run |
| image_pull_policy | Always / IfNotPresent |
| content | base64 encoded tar.gz of injectable content, to overlay on the container prior to execution |
| entrypoint | Array of strings defining the entrypoint in the container, see `docker run`|
| run | Array of strings defining the command to run, see `docker run`|
| working_directory | Working directory for the command to run in |
| env_vars | Environment variables passed to the job container |
| secret_refs | Array of strings `variable_name@vault_path` to retrieve from the Vault and inject into the container as /secrets.yml \| .json |
| secret_file_type | String 'yaml' or 'json' |
| cont_on_warnings | Boolean default false. Whether to panic or continue on secret resolving warnings from the Vault |

These are the additional fields returned:

| Parameter | |
|-----------|-|
| status | Current status of the job |
| return_code | Return code from the job container |
| submitted | When submitted |
| started |  When started |
| ended | When ended |
| output | Raw output from the job (stdout+stderr) |
| container_id | docker container id |
| kill_requested | Boolean kill has been requested |
