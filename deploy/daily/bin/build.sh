#!/bin/bash

main_dir=/Users/daijia/go/src/x/src/gx/main.go

rm -f ./rocket_node
go clean
go build  -o ./rocket_node $main_dir
