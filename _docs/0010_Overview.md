---
title: Overview
classes: wide
toc: true
---
## GoStint
GoStint is a [MVP] job submission and execution API where all jobs run in their
own docker containers.  Authentication, end-to-end encryption and
tamper-proofing of the secure protocol
are implemented via integration with [Hashicorp Vault](https://www.vaultproject.io/),
utilising vault features:

* AppRole Authentication
* Token Generation
* Transit Encryption
* Cubbyholes
* Response Wrapping
* Database Credential Management
* Secret Engines

Leveraging the Vault Secret Engines, GoStint can inject secrets into the container
for the job.

Submitting a job to GoStint involves:

1. Authenticating with the Vault (using either a Vault Token or AppRole),
2. Getting a Response Wrapped AppRole Secret_Id for the GoStint service.
3. Encrypting the job request using Vault's Transit plugin.
4. Putting the encrypted job in a single-use response-wrapped Vault Cubbyhole.
5. Submitting a job wrapper that tells GoStint where to get the payload from in the Vault.
6. Polling the status of the job until complete.

An example script performing the above steps can be seen
[Running a Job - the hard way]({{ site.baseurl }}{% link _docs/0110_Running_A_Job.md %}).

However, to make things easier, a sister project called
[gostint-client](https://github.com/goethite/gostint-client)
is available providing a command-line tool and golang api that does all of the above steps
for you. See
[Running a Job - the easy way with gostint-client]({{ site.baseurl }}{% link _docs/0120_Running_A_Job_client.md %}).

### Jobs as Containers
Running jobs in containers allows the exact versioned environment to be (re)used
for each job. Here are some examples:

* Ansible playbooks that have been developed against a specific version of Ansible
  can be ensured to be run in production with that exact same version (including any/all
  dependencies).
* Terraform infrastructure-as-code projects, again run in a container with the exact
  same tested version of Terraform.
* Kubectl / Helm charts with a KUBECONFIG injected from Vault and ensured to run
  with the correct versions of the tools.
* Powershell scripts (in Ubuntu containers) for windows automations.

Basically anything that can be run as a job in a container can be securely driven
by the GoStint API.

### Jobs in Queues
GoStint serialises consecutive jobs in queues (using `qname`). Queues are entirely
arbitrary and you can use them in whatever way suits your purposes.
The default queue is "".

Note: There is currently no limit to how many jobs can run in parallel (if you
let them) - other than the resource limitations of the host, of course.

### Layering Content
The `-content=folder/` option allows for additional content to be layered on top
of the docker container, prior to running it.  The layer is unpacked in to the
container routed at "`/`".

### Injecting Secrets from Vault
The `-secret-refs=["variable_name@secret/data/mysecrets.myvalue1", ...]` allows for variables
to be set from paths in the Vault.  These are injected at the moment the job's docker
container is instantiated and placed in the container either as `/secrets.yml` or
`/secrets.json`  (depending on `-secret-filetype` - default is `yaml`).

### A Helm Chart for Kubernetes
A demonstrator [Helm Chart](https://github.com/goethite/gostint-helm)
is available to deploy GoStint, Vault with etcd
backend, and MongoDB - as a self-contained automation API.

Note: The gostint pods are run in "privileged" mode to enable
support for docker-in-docker running of the containerised jobs.

### Project Status
It is early days for this project and it is still considered a MVP.

### Job States
![jobstates](https://raw.githubusercontent.com/goethite/gostint/master/docs/jobstates.mermaid.png)

### Job Sequence Diagram
This diagram is taken from the original brainstorming for the end-to-end
secure job submission design - you can see the full document
[here](https://github.com/goethite/gostint/blob/master/docs/jobsequence.md).
This is now fully implemented between
[gostint-client](https://github.com/goethite/gostint-client)
and
[gostint](https://github.com/goethite/gostint)

![jobsequence](https://raw.githubusercontent.com/goethite/gostint/master/docs/job_via_intermediary.mermaid.png)
