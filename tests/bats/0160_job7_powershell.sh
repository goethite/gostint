#!/usr/bin/env bats

@test "Submitting job7 powershell should return json" {
  # Get secretId for the approle
  SECRETID=$(vault write -f auth/approle/role/goswim-role/secret-id | grep "^secret_id " | awk '{ print $2; }')
  echo "$SECRETID" > $BATS_TMPDIR/secretid

  J="$(curl -k -s https://127.0.0.1:3232/v1/api/job --header "X-Secret-Token: $SECRETID" -X POST -d @../job7_powershell.json | tee $BATS_TMPDIR/job7.json)"
  echo $J >&2
  [ "$J" != "" ]
}

@test "job7 should be queued in the powershell_q queue" {

  J="$(cat $BATS_TMPDIR/job7.json)"
  echo $J >&2

  id=$(echo $J | jq ._id -r)
  status=$(echo $J | jq .status -r)
  qname=$(echo $J | jq .qname -r)

  [ "$id" != "" ] && [ "$status" == "queued" ] && [ "$qname" == "powershell_q" ]
}

@test "Be able to retrieve the current status" {
  SECRETID="$(cat $BATS_TMPDIR/secretid)"
  J="$(cat $BATS_TMPDIR/job7.json)"

  ID=$(echo $J | jq ._id -r)

  R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID --header "X-Secret-Token: $SECRETID")"
  echo "R:$R" >&2
  status=$(echo $R | jq .status -r)

  [ "$status" == "queued" -o "$status" == "running" ]
}

@test "Status should eventually be success" {
  SECRETID="$(cat $BATS_TMPDIR/secretid)"
  J="$(cat $BATS_TMPDIR/job7.json)"

  ID=$(echo $J | jq ._id -r)
  echo "ID:$ID" >&2

  status="queued"
  for i in {1..40}
  do
    sleep 5
    R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID --header "X-Secret-Token: $SECRETID")"
    echo "R:$R" >&2
    status=$(echo $R | jq .status -r)
    if [ "$status" != "queued" -a "$status" != "running" ]
    then
      break
    fi
  done
  echo "status after:$status" >&2
  echo "$R" > $BATS_TMPDIR/job7.final.json
  [ "$status" == "success" ]
}

@test "Should have final output in json, with injected secret" {
  R="$(cat $BATS_TMPDIR/job7.final.json)"

  echo "R:$R" >&2
  output="$(echo $R | jq .output -r)"

  # [ "$output" != "" ]
  echo "$output" | grep "mysecret : s3cr3t"
}

@test "Should delete the job id" {
  SECRETID="$(cat $BATS_TMPDIR/secretid)"
  J="$(cat $BATS_TMPDIR/job7.json)"

  ID=$(echo $J | jq ._id -r)
  echo "ID:$ID" >&2

  R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID -X DELETE --header "X-Secret-Token: $SECRETID")"
  echo "R:$R" >&2

  DELID=$(echo "$R" | jq ._id -r)

  [ "$DELID" == "$ID" ]
}

@test "Lookup for deleted id should return Not Found error" {
  SECRETID="$(cat $BATS_TMPDIR/secretid)"
  J="$(cat $BATS_TMPDIR/job7.json)"

  ID=$(echo $J | jq ._id -r)
  echo "ID:$ID" >&2

  R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/$ID --header "X-Secret-Token: $SECRETID")"
  echo "R:$R" >&2

  echo "$R" | grep "Not Found"
}

@test "Lookup for garbage id should return Invalid job ID error" {
  SECRETID="$(cat $BATS_TMPDIR/secretid)"
  J="$(cat $BATS_TMPDIR/job7.json)"

  R="$(curl -k -s https://127.0.0.1:3232/v1/api/job/DOESNOTEXIST --header "X-Secret-Token: $SECRETID")"
  echo "R:$R" >&2

  echo "$R" | grep "Invalid job ID"
}
