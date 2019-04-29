#!/bin/bash

for file in ./pid/pid_gx*
do
    kill -9 `cat $file`
    rm -f $file
done

docker stop $(docker ps -q)
docker container prune