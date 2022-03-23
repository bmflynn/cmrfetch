#!/bin/bash
set -e
export VER=`git describe --dirty`
export CGO_ENABLED=0

for os in linux windows darwin; do
    export GOOS=${os}
    for arch in amd64; do
        export GOARCH=${arch}
        bin="cmrsearch_${os}_${arch}"
        if [[ ${os} == "windows" ]]; then
            bin=${bin}.exe
        fi
        echo "building ${bin}"
        go build -o ${bin} -ldflags "-s -w -X main.version=${VER} --extldflags '-static'"
    done
done
