#!/usr/bin/env bats

@test "Submitting job4 shell long sleep for kill test should return json" {
  # Get secretId for the approle
  SECRETID=$(vault write -f auth/approle/role/goswim-role/secret-id | grep "^secret_id " | awk '{ print $2; }')
  echo "$SECRETID" > $BATS_TMPDIR/secretid

  J="$(curl -k -s https://127.0.0.1:3232/v1/api/job --header "X-Secret-Token: $SECRETID" -X POST -d @../job4_sleep.json | tee $BATS_TMPDIR/job4.json)"
  echo "J=$J" >&2
  [ "$J" != "" ]
}

@test "job4 should be queued in the play job4 queue" {

  J="$(cat $BATS_TMPDIR/job4.json)"
  echo "J=$J" >&2

  id=$(echo $J | jq ._id -r)
  status=$(echo $J | jq .status -r)
  qname=$(echo $J | jq .qname -r)

  [ "$id" != "" ] && [ "$status" == "queued" ] && [ "$qname" == "play job4" ]
}

@test "Be able to retrieve the current status" {
  SECRETID="$(cat $BATS_TMPDIR/secretid)"
  J="$(cat $BATS_TMPDIR/job4.json)"

  ID=$(echo $J | jq ._id -r)

  R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID --header "X-Secret-Token: $SECRETID")"
  echo "R:$R" >&2
  status=$(echo $R | jq .status -r)

  [ "$status" == "queued" -o "$status" == "running" ]
}

@test "Status should eventually be running" {
  SECRETID="$(cat $BATS_TMPDIR/secretid)"
  J="$(cat $BATS_TMPDIR/job4.json)"

  ID=$(echo $J | jq ._id -r)
  echo "ID:$ID" >&2

  status="queued"
  for i in {1..20}
  do
    sleep 1
    R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID --header "X-Secret-Token: $SECRETID")"
    echo "R:$R" >&2
    status=$(echo $R | jq .status -r)
    container_id=$(echo $R | jq .container_id -r)
    if [ "$status" != "queued" ]
    then
      if [ "$status" != "running" ]
      then
        break
      elif [ "$container_id" != "" ]
      then
        break
      fi
    fi
  done
  echo "status after:$status" >&2
  echo "$R" > $BATS_TMPDIR/job4.final.json
  [ "$status" == "running" ]
}

@test "Be able to send a kill to the job" {
  SECRETID="$(cat $BATS_TMPDIR/secretid)"
  J="$(cat $BATS_TMPDIR/job4.json)"

  ID=$(echo $J | jq ._id -r)

  R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/kill/$ID -X POST --header "X-Secret-Token: $SECRETID")"
  echo "R:$R" >&2
  status=$(echo $R | jq .status -r)
  kill_requested=$(echo $R | jq .kill_requested -r)

  [ "$status" == "running" -a "$kill_requested" == "true" ]
}

@test "Status should eventually be stopping or failed" {
  SECRETID="$(cat $BATS_TMPDIR/secretid)"
  J="$(cat $BATS_TMPDIR/job4.json)"

  ID=$(echo $J | jq ._id -r)
  echo "ID:$ID" >&2

  status="running"
  for i in {1..20}
  do
    sleep 5
    R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID --header "X-Secret-Token: $SECRETID")"
    echo "R:$R" >&2
    status=$(echo $R | jq .status -r)
    if [ "$status" != "running" ]
    then
      break
    fi
  done
  echo "status after:$status" >&2
  echo "$R" > $BATS_TMPDIR/job4.final.json
  [ "$status" == "stopping" -o "$status" == "failed" ]
}

@test "Status should eventually be failed" {
  SECRETID="$(cat $BATS_TMPDIR/secretid)"
  J="$(cat $BATS_TMPDIR/job4.json)"

  ID=$(echo $J | jq ._id -r)
  echo "ID:$ID" >&2

  status="stopping" # or failed, see above
  for i in {1..20}
  do
    sleep 5
    R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID --header "X-Secret-Token: $SECRETID")"
    echo "R:$R" >&2
    status=$(echo $R | jq .status -r)
    if [ "$status" != "stopping" ]
    then
      break
    fi
  done
  echo "status after:$status" >&2
  echo "$R" > $BATS_TMPDIR/job4.final.json
  [ "$status" == "failed" ]
}
