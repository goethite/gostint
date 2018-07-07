#!/usr/bin/env bats

@test "Submitting job1 should return json" {
  # Get secretId for the approle
  SECRETID=$(vault write -f auth/approle/role/goswim-role/secret-id | grep "^secret_id " | awk '{ print $2; }')
  echo "$SECRETID" > $BATS_TMPDIR/secretid

  J="$(curl -s http://127.0.0.1:3232/v1/api/job --header "X-Secret-Token: $SECRETID" -X POST -d @../job1.json | tee $BATS_TMPDIR/job1.json)"
  [ "$J" != "" ]
}

@test "job1 should be queued in the play queue" {

  J="$(cat $BATS_TMPDIR/job1.json)"

  id=$(echo $J | jq ._id -r)
  status=$(echo $J | jq .status -r)
  qname=$(echo $J | jq .qname -r)

  [ "$id" != "" ] && [ "$status" == "queued" ] && [ "$qname" == "play" ]
}

@test "Be able to retrieve the current status" {
  SECRETID="$(cat $BATS_TMPDIR/secretid)"
  J="$(cat $BATS_TMPDIR/job1.json)"

  ID=$(echo $J | jq ._id -r)

  R="$(curl -s http://127.0.0.1:3232/v1/api/job/$ID --header "X-Secret-Token: $SECRETID")"
  echo "R:$R" >&2
  status=$(echo $J | jq .status -r)

  [ "$status" == "queued" ]
}

@test "Status should eventually be success" {
  SECRETID="$(cat $BATS_TMPDIR/secretid)"
  J="$(cat $BATS_TMPDIR/job1.json)"

  ID=$(echo $J | jq ._id -r)
  echo "ID:$ID" >&2

  status="queued"
  for i in {1..5}
  do
    sleep 5
    R="$(curl -s http://127.0.0.1:3232/v1/api/job/$ID --header "X-Secret-Token: $SECRETID")"
    echo "R:$R" >&2
    status=$(echo $R | jq .status -r)
    if [ "$status" != "queued" ]
    then
      break
    fi
  done
  echo "status after:$status" >&2
  echo "$R" > $BATS_TMPDIR/job1.final.json
  [ "$status" == "success" ]
}

@test "Should have final output in json" {
  R="$(cat $BATS_TMPDIR/job1.final.json)"

  echo "R:$R" >&2
  output="$(echo $R | jq .output -r)"

  [ "$output" != "" ]
}
