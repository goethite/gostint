#!/usr/bin/env bats

@test "Submitting job2 ansible ping should return json" {
  # Get a default token for the api post authentication
  TOKEN=$(vault write -f auth/token/create policies=default -format=json | jq .auth.client_token -r)
  echo "$TOKEN"
  echo "$TOKEN" > $BATS_TMPDIR/token

  # Get secretId for the approle
  SECRETID=$(vault write -f auth/approle/role/goswim-role/secret-id | grep "^secret_id " | awk '{ print $2; }')
  echo "$SECRETID" > $BATS_TMPDIR/secretid

  cat ../job2_ansible.json | jq ".secret_id=\"$SECRETID\"" > $BATS_TMPDIR/job.json

  J="$(curl -k -s https://127.0.0.1:3232/v1/api/job --header "X-Auth-Token: $TOKEN" -X POST -d @$BATS_TMPDIR/job.json | tee $BATS_TMPDIR/job2.json)"
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
  # SECRETID="$(cat $BATS_TMPDIR/secretid)"
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
  # SECRETID="$(cat $BATS_TMPDIR/secretid)"
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
