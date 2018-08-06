---
title: Use Cases
classes: wide
toc: true
---

# GoStint Use Cases

# Single Tenant
![Single Tenant](../diagrams/usecase_single_tenant.png)

TODO: Authentication mechanisms for this use case.

# Multi Tenant
![Single Tenant](../diagrams/usecase_multi_tenant.png)

TODO: Authentication mechanisms for this use case.

# Job Sequence Diagram
{% mermaid %}
sequenceDiagram
  participant requestor
  participant poster
  participant o as orchestrator
  participant gostint
  participant queues
  participant vault as vault
  participant docker

  %% Enrolement
  o->>vault: Onboards requestor
  o->>vault: Onboards gostint's transit keyring, policies, etc...
  %% requestor can only encrypt.
  %% gostint can also decrypt.
  %% poster, if even in vault at all, has no permissions here.

  %% build job to submit
  requestor->>vault: (authenticates with)
  requestor->>vault: request wrapped SecretID for AppRole(gostint)
  vault-->>requestor: wrapped SecretID (token)

  requestor->>vault: request transit keyring to encrypt job payload
  %% the plaintext sent is a base64 encoded json document
  vault-->>requestor: cyphertext

  requestor->>vault: request a default token ttl=10m use-limit=2
  vault-->>requestor: a default token
  requestor->>vault: place encrypted job payload in the default token's cubbyhole

  %% request job to be posted/routing
  requestor->>poster: (authenticates with)
  requestor->>poster: POST job qname+default token+cubbyhole path+wrapped SecretID

  %% This time, even if the poster is hacked and intercepts the POST request,
  %% and using the default token to retrieve the cubbyhole, the data returned
  %% is encrypted, such that only gostint's transit keyring can decrypt it.
  %% This tampering of the request can be detected to raise an alert of the
  %% MITM attack.

  poster->>gostint: (authenticates with)
  poster->>gostint: fwd POST job request

  %% extract job from cubbyhole
  gostint->>vault: retrieve cubbyhole from path using default token (last use)
  vault-->>gostint: (still encrypted) job request from cubbyhole

  %% we can decrypt here or at point of job execution, in this example we will
  %% leave the payload encrypted until it is needed for the job to run.

  gostint->>queues: Queues the (still encrypted) job request
  %% Note; the wrapped SecretID is not encrypted
  gostint-->>poster: job queued response
  poster-->>requestor: job queued response

  gostint-->>gostint: sometime later

  queues->>gostint: job is popped from the queue

  %% authenticate
  gostint->>vault: unwrap wrapped SecretID
  vault-->>gostint: SecretID
  gostint->>vault: authenticate with RoleID+SecretID
  vault-->>gostint: token (with appropriate policies for automation)
  %% this token is used by gostint going fwd and passed to running job

  %% Decrypt
  %% gostint->>gostint: decrypt payload with RSA private key
  gostint->>vault: request decrypt of job using keyring
  vault-->>gostint: plaintext base64 encoded job request payload

  gostint->>vault: retrieve secrets at refs from job request
  vault-->>gostint: secrets

  gostint->>docker: run job request with injected secrets...
  docker-->>gostint: return results
  gostint->>queues: save results
  gostint->>vault: revoke approle token (drop job privs)

  requestor->>poster: poll for results
  poster->>gostint: poll for results
  gostint->> queues: retrieve results
  queues-->>gostint: results
  gostint-->>poster: results&#xb3;
  poster-->>requestor: results

  requestor-->>requestor: loop polls until success/failed/notauthorised/unknown

{% endmermaid %}
