#!/bin/bash

for file in /home/x/run/pid/pid_gx*
do
    kill -9 `cat $file`
    rm -f $file
done
