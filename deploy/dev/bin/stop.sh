#!/bin/bash

cd /home/rocket_node_dev/run
for file in pid/pid_gx*
do
    kill -9 `cat $file`
    rm -f $file
done
