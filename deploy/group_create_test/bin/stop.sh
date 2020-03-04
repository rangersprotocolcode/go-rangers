#!/bin/bash

cd /home/group_create_test_x/run
for file in pid/pid_gx*
do
    kill -9 `cat $file`
    rm -f $file
done
