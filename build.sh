#!/bin/bash
set -e
export VER=`git describe --dirty`
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
go build -a -ldflags "-s -w -X main.version=${VER} --extldflags '-static'"
