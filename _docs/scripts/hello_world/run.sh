#!/bin/bash -e

export VAULT_ADDR=http://127.0.0.1:8300

vault login root

# Get a default token for the api post authentication
TOKEN=$(vault write -f auth/token/create policies=default \
  -format=json | jq .auth.client_token -r)

# Get secretId for the approle
WRAPSECRETID=$(vault write -wrap-ttl=144h -f \
  auth/approle/role/gostint-role/secret-id -format=json \
  | jq .wrap_info.token -r)

# encrypt job payload using vault transit secret engine
B64=$(base64 < hello_world_job.json)
E=$(vault write transit/encrypt/gostint plaintext="$B64" \
  -format=json | jq .data.ciphertext -r)

# Put encrypted payload in a cubbyhole of an ephemeral token
CUBBYTOKEN=$(vault token create -policy=default -ttl=60m \
  -use-limit=2 -format=json | jq .auth.client_token -r)
VAULT_TOKEN=$CUBBYTOKEN vault write cubbyhole/job payload="$E"

# Get qname for job wrapper json
QNAME=$(cat hello_world_job.json | jq .qname -r)

# Create new job request with encrypted payload
JOB_WRAP_JSON=$(jq --arg qname "$QNAME" \
  --arg cubby_token "$CUBBYTOKEN" \
  --arg cubby_path "cubbyhole/job" \
  --arg wrap_secret_id "$WRAPSECRETID" \
  '. | .qname=$qname | .cubby_token=$cubby_token
    | .cubby_path=$cubby_path
    | .wrap_secret_id=$wrap_secret_id' \
  <<<'{}')

echo "Submitting wrapped job:"
echo $JOB_WRAP_JSON | jq .

RES=$(curl -k -s https://127.0.0.1:3232/v1/api/job \
  --header "X-Auth-Token: $TOKEN" \
  -X POST \
  -d "$(echo $JOB_WRAP_JSON)")
echo "Results of job submitted to queue:"
jq . <<<$RES

# Get the ID of the submitted job
ID=$(echo $RES | jq ._id -r)

# Loop until complete of failed
status="queued"
for i in {1..20}
do
  sleep 1
  R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID \
    --header "X-Auth-Token: $TOKEN")"
  jq . <<<$R
  status=$(echo $R | jq .status -r)
  if [ "$status" != "queued" -a "$status" != "running" ]
  then
    break
  fi
done
echo "final status:$status" >&2
echo "Output of the job:"
jq .output -r <<<$R
