#!/bin/bash

cd /home/x/run
for file in pid/pid_gx*
do
    kill -9 `cat $file`
    rm -f $file
done