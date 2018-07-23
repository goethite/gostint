# goswim job states

```mermaid
graph TD
  POST((POST)) -- submits job --> queued;
  queued == "popped off queue (atomic op)" ==> running;
  running == authentication failed ==> notauthorised[fa:fa-ban notauthorised];
  running == job failed rc!=0 ==> failed[fa:fa-times failed];
  running == job completes rc=0 ==> success[fa:fa-check success];
  running == kill requested ==> stopping;
  running == goswim node failed ==> unknown[fa:fa-question unknown];
  stopping ==> failed;

  style queued fill:#8cf
  style running fill:#8af
  style stopping fill:#fa8
  style failed fill:#f88
  style notauthorised fill:#f88
  style unknown fill:#f35
  style success fill:#0b0

fini((end));
  notauthorised --> fini;
  failed --> fini;
  success --> fini;
  unknown --> fini;

```
