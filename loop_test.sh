#!/bin/bash -e

CNT=0
RC=0
while [ $RC = 0 ]
do
  let CNT=$CNT+1
  echo "Iteration: $CNT"
  godo test
  RC=$?
done
