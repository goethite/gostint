# Job Sequence Diagram

```mermaid
sequenceDiagram
  participant requestor
  participant poster
  participant goswim
  participant vault
  participant queues
  participant docker

  requestor->>poster: submit job to run with secretId[1]
  poster->>goswim: POST job with secretId[2]
  goswim->>vault: authenticate poster (approle: secretId[2])
  vault-->>goswim: token[1] (discarded/revoked)
  goswim->>queues: push to a queue
  queues->>goswim: pop next from a queue
  goswim->>vault: authenticate requestor (approle: secretId[1])
  vault-->>goswim: token[2]
  goswim->>vault: get requested secrets for job
  goswim->>docker: runs requested job with secrets from vault
  docker-->>goswim: job completes
  goswim->>queues: job status/results are saved
  goswim->>vault: revoke token[1]

  requestor->>poster: polls for results
  poster->>goswim: get results for job
  goswim->>queues: get job results
  queues-->>goswim: return job results
  goswim-->>poster: return job results
  poster-->>requestor: return job results
```
* Requestor and poster can be the same enitity, in which case secretId[1] and [2]
are the same.
* This two step authentication of requestor and poster allows for an intermediary
api routing "middleware" - in future this will be leveraged to support the
requestor posting the job details into a Vault "Cubbyhole" for goswim to pickup,
thereby removing the risk of a man-in-the-middle-attack.
