#!/bin/bash
set -e
export VER=`git describe --dirty`
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

function build() {
  (
    export GOOS=$1
    export GOARCH=$2
    local sfx=${GOOS}_${GOARCH}
    if [[ $GOOS == "windows" ]]; then
      sfx=${sfx}.exe
    fi
    pkg=github.com/bmflynn/cmrfetch/internal
    buildtime=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    sha=$(git rev-parse HEAD)
    go build \
      -o ./build/cmrfetch.${sfx} \
      -ldflags "-s -w -X ${pkg}.Version=${VER} -X ${pkg}.BuildTime=${buildtime} -X ${pkg}.GitSHA=${sha} --extldflags '-static'"
  )
}

mkdir -pv build
build linux amd64
build windows amd64
build darwin amd64
build darwin arm64
