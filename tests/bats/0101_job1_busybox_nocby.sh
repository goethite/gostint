#!/usr/bin/env bats

@test "Submitting job1 busybox without cubbyhole should return json" {
  # Get a default token for the api post authentication
  TOKEN=$(vault write -f auth/token/create policies=default -format=json | jq .auth.client_token -r)
  echo "TOKEN: $TOKEN" >&2
  echo "$TOKEN" > $BATS_TMPDIR/token

  # Get secretId for the approle
  WRAPSECRETID=$(vault write -wrap-ttl=144h -f auth/approle/role/$GOSTINT_ROLENAME/secret-id -format=json | jq .wrap_info.token -r)
  echo "WRAPSECRETID: $WRAPSECRETID" >&2

  QNAME=$(cat ../job1.json | jq .qname -r)

  # Create new job request with the wrapped secret id
  jq --arg wrap_secret_id "$WRAPSECRETID" \
    '. | .wrap_secret_id=$wrap_secret_id' \
    < ../job1.json >$BATS_TMPDIR/job_ncby.json
  cat $BATS_TMPDIR/job_ncby.json >&2

  J="$(curl -k -s https://127.0.0.1:3232/v1/api/job --header "X-Auth-Token: $TOKEN" -X POST -d @$BATS_TMPDIR/job_ncby.json | tee $BATS_TMPDIR/job1_ncby.json)"
  echo "J: $J" >&2
  [ "$J" != "" ]
}

@test "job1 should be queued in the play job1 queue" {

  J="$(cat $BATS_TMPDIR/job1_ncby.json)"
  echo "J: $J" >&2

  id=$(echo $J | jq ._id -r)
  status=$(echo $J | jq .status -r)
  qname=$(echo $J | jq .qname -r)

  [ "$id" != "" ] && [ "$status" == "queued" ] && [ "$qname" == "play job1" ]
}

@test "Be able to retrieve the current status" {
  TOKEN="$(cat $BATS_TMPDIR/token)"
  echo "TOKEN: $TOKEN" >&2
  J="$(cat $BATS_TMPDIR/job1_ncby.json)"

  ID=$(echo $J | jq ._id -r)

  R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID --header "X-Auth-Token: $TOKEN")"
  echo "R:$R" >&2
  status=$(echo $R | jq .status -r)

  [ "$status" == "queued" -o "$status" == "running" ]
}

@test "Status should eventually be success" {
  TOKEN="$(cat $BATS_TMPDIR/token)"
  echo "TOKEN: $TOKEN" >&2
  J="$(cat $BATS_TMPDIR/job1_ncby.json)"

  ID=$(echo $J | jq ._id -r)
  echo "ID:$ID" >&2

  status="queued"
  for i in {1..20}
  do
    sleep 1
    R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID --header "X-Auth-Token: $TOKEN")"
    echo "R:$R" >&2
    status=$(echo $R | jq .status -r)
    if [ "$status" != "queued" -a "$status" != "running" ]
    then
      break
    fi
  done
  echo "status after:$status" >&2
  echo "$R" > $BATS_TMPDIR/job1_ncby.final.json
  [ "$status" == "success" ]
}

@test "Should have final output in json" {
  R="$(cat $BATS_TMPDIR/job1_ncby.final.json)"

  echo "R:$R" >&2
  output="$(echo $R | jq .output -r)"

  [ "$output" != "" ]
}

@test "Should delete the job id" {
  TOKEN="$(cat $BATS_TMPDIR/token)"
  echo "TOKEN: $TOKEN" >&2
  J="$(cat $BATS_TMPDIR/job1_ncby.json)"

  ID=$(echo $J | jq ._id -r)
  echo "ID:$ID" >&2

  R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID -X DELETE --header "X-Auth-Token: $TOKEN")"
  echo "R:$R" >&2

  DELID=$(echo "$R" | jq ._id -r)

  [ "$DELID" == "$ID" ]
}

@test "Lookup for deleted id should return Not Found error" {
  TOKEN="$(cat $BATS_TMPDIR/token)"
  echo "TOKEN: $TOKEN" >&2
  J="$(cat $BATS_TMPDIR/job1_ncby.json)"

  ID=$(echo $J | jq ._id -r)
  echo "ID:$ID" >&2

  R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID --header "X-Auth-Token: $TOKEN")"
  echo "R:$R" >&2

  echo "$R" | grep "Not Found"
}

@test "Lookup for garbage id should return Invalid job ID error" {
  TOKEN="$(cat $BATS_TMPDIR/token)"
  echo "TOKEN: $TOKEN" >&2
  J="$(cat $BATS_TMPDIR/job1_ncby.json)"

  R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/DOESNOTEXIST --header "X-Auth-Token: $TOKEN")"
  echo "R:$R" >&2

  echo "$R" | grep "Invalid job ID"
}
