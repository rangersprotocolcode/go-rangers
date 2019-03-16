#!/bin/bash

main_dir=../../src/gx/main.go

rm -f ./gx
go clean
go build  -o ./gx $main_dir








