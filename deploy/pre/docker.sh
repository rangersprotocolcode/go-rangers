#!/bin/bash

image_name=$1
file_name=$2

docker stop $(docker ps -q);
docker container prune;
docker rmi $(docker images | grep $image_name | awk '{print $3}');
docker rmi $(docker images | grep "none" | awk '{print $3}');
cd /home/x;
docker load < $file_name;