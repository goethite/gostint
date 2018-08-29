---
title: Overview
classes: wide
toc: true
---
## GoStint
GoStint is a [proof-of-concept] job submission and execution API where all jobs run in their
own docker containers.  All authentication, encryption of the secure protocol
are implemented via integration with Hashicorp Vault - utilising vault features:

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
is available providing a command-line tool that does all of the above steps
for you. See
[Running a Job - the easy way with gostint-client]({{ site.baseurl }}{% link _docs/0120_Running_A_Job_client.md).

### Jobs as Containers
Running jobs in containers allows the exact versioned environment to be (re)used
for each job. Here are some examples:

* Ansible playbooks that have been developed against a specific version of Ansible
  can be ensured to be run in production with that exact same version (including any/all
  dependencies).
* Terraform infrastructure-as-code projects, again run in a container with the exact
  same tested version of terraform.
* Kubectl / Helm charts with a KUBECONFIG injected from Vault and ensured to run
  with the correct versions of the tools.
* Powershell scripts (in Ubuntu containers) for windows automations.

Basically anything that can be run as a job in a container can be securely driven
by the GoStint API.

### A Helm Chart for Kubernetes
A [proof-of-concept] [Helm Chart](https://github.com/goethite/gostint-helm)
is available to deploy GoSting, Vault with etcd
backend, and MongoDB - as a self-contained automation API.

Note: The job gostint pods are currently run in "privileged" mode to enable
support for docker-in-docker running of the containerised jobs.

### Project Status
It is early days for this project and it is still considered a proof-of-concept,
and would certainly not recommend for production at this stage.
More work needs to be done around reviewing and securing the api protocols.
