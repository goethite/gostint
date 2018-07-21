#!/bin/bash -e

echo
echo "***************************"
echo "*** Starting BATS Tests ***"
echo "***************************"
echo

vault login root

mongo admin -u goswim_admin -p admin123 --eval "db=db.getSiblingDB('goswim'); db.queues.remove({})"
