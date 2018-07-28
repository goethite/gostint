#!/bin/sh -x
LANG=C.UTF-8

echo "Hello World!"
env
ps -efl

ls -l /

ls -laR /gostint /secrets*
cat /secrets.yml

#ping -c 3 www.google.com  needs root/sudo/u+s

pwd
