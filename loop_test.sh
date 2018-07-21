#!/bin/bash -e

CNT=0
while
  godo test
  let CNT=$CNT+1
  echo "Iteration: $CNT"
do
  sleep 1
done
