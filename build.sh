#!/bin/bash
set -e
export VER=`git describe --dirty`
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

function build() {
  export GOOS=$1
  export GOARCH=$2
  local sfx=${GOOS}_${GOARCH}
  pkg=github.com/bmflynn/cmrfetch/internal
  buildtime=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  sha=$(git rev-parse HEAD)

  binname=cmrfetch
  if [[ $GOOS == "windows" ]]; then
    binname=${binname}.exe
  fi
  outdir=./build/cmrfetch_${sfx}
  mkdir -p ${outdir}
  go build \
    -o ${outdir}/${binname} \
    -ldflags "-s -w -X ${pkg}.Version=${VER} -X ${pkg}.BuildTime=${buildtime} -X ${pkg}.GitSHA=${sha} --extldflags '-static'"
  cp README.md ${outdir}
  (
    cd build
    tar -cvzf cmrfetch_${sfx}.tar.gz $(basename ${outdir})
  )
  rm -rf ${outdir}
}

mkdir -pv build
build linux amd64
build windows amd64
build darwin amd64
build darwin arm64
