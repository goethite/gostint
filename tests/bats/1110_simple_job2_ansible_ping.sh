#!/usr/bin/env bats

@test "Simple API - Submitting job2 ansible ping should return json" {
  # Get a default token for the api post authentication
  TOKEN=$(
    vault write -f \
      auth/token/create \
      policies=default \
      -format=json \
      | jq .auth.client_token -r
  )
  echo "$TOKEN"
  echo "$TOKEN" > $BATS_TMPDIR/token

  # Get secretId for the approle
  WRAPSECRETID=$(
    vault write -wrap-ttl=144h \
      -f auth/approle/role/$GOSTINT_ROLENAME/secret-id \
      -format=json \
      | jq .wrap_info.token -r
  )
  echo "WRAPSECRETID: $WRAPSECRETID" >&2
  # echo "$WRAPSECRETID" > $BATS_TMPDIR/wrapsecretid

    # Create new job request with payload
  jq --arg wrap_secret_id "$WRAPSECRETID" \
     '. | .wrap_secret_id=$wrap_secret_id' \
     < ../job2_ansible.json >$BATS_TMPDIR/job.json
  cat $BATS_TMPDIR/job.json >&2

  J="$(
    curl -k -s https://127.0.0.1:3232/v1/api/job \
      --header "X-Auth-Token: $TOKEN" \
      -X POST \
      -d @$BATS_TMPDIR/job.json \
      | tee $BATS_TMPDIR/job2.json
  )"
  [ "$J" != "" ]
}

@test "job2 should be queued in the play job2 queue" {

  J="$(cat $BATS_TMPDIR/job2.json)"

  id=$(echo $J | jq ._id -r)
  status=$(echo $J | jq .status -r)
  qname=$(echo $J | jq .qname -r)

  [ "$id" != "" ] && [ "$status" == "queued" ] && [ "$qname" == "play job2" ]
}

@test "Be able to retrieve the current status" {
  TOKEN="$(cat $BATS_TMPDIR/token)"
  echo "TOKEN: $TOKEN" >&2
  J="$(cat $BATS_TMPDIR/job2.json)"

  ID=$(echo $J | jq ._id -r)

  R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID --header "X-Auth-Token: $TOKEN")"
  echo "R:$R" >&2
  status=$(echo $R | jq .status -r)

  [ "$status" == "queued" -o "$status" == "running" ]
}

@test "Status should eventually be success" {
  TOKEN="$(cat $BATS_TMPDIR/token)"
  echo "TOKEN: $TOKEN" >&2
  J="$(cat $BATS_TMPDIR/job2.json)"

  ID=$(echo $J | jq ._id -r)
  echo "ID:$ID" >&2

  status="queued"
  for i in {1..40}
  do
    sleep 5
    R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID --header "X-Auth-Token: $TOKEN")"
    echo "R:$R" >&2
    status=$(echo $R | jq .status -r)
    if [ "$status" != "queued" -a "$status" != "running" ]
    then
      break
    fi
  done
  echo "status after:$status" >&2
  echo "$R" > $BATS_TMPDIR/job2.final.json
  [ "$status" == "success" ]
}

@test "Should have final output in json" {
  R="$(cat $BATS_TMPDIR/job2.final.json)"

  echo "R:$R" >&2
  output="$(echo $R | jq .output -r)"

  echo "$output" | grep "pong"
}
