#!/bin/bash
#
# Build script for local or dev builds. Production or release builds should be 
# handled via the SLA Go Releaser GitHub Action.
#
set -e
export builddir=./build

commit_date=$(git log --date=iso8601-strict -1 --pretty=%ct)
commit=$(git rev-parse HEAD)
version=$(git describe --tags --always --dirty | cut -c2-)
tree_state=$(if git diff --quiet; then echo "clean"; else echo "dirty"; fi)

function build() {
  export GOOS=$1
  export GOARCH=$2
  export CGO_ENABLED=0

  pkg=github.com/bmflynn/cmrfetch/internal
  buildtime=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  sha=$(git rev-parse HEAD)

  binname=cmrfetch-${GOOS}-${GOARCH}
  if [[ $GOOS == "windows" ]]; then
    binname=${binname}.exe
  fi
    
  ldflags="-s -w --extldflags '-static'"
  ldflags="${ldflags} -X github.com/bmflynn/cmrfetch/internal.Version=${version}"
  ldflags="${ldflags} -X github.com/bmflynn/cmrfetch/internal.Commit=${commit}"
  ldflags="${ldflags} -X github.com/bmflynn/cmrfetch/internal.CommitDate=${commit_date}"
  ldflags="${ldflags} -X github.com/bmflynn/cmrfetch/internal.TreeState=${tree_state}"
  go build -o ${builddir}/${binname} -ldflags "${ldflags}"
  #  -ldflags "-s -w -X ${pkg}.Version=${VER} -X ${pkg}.BuildTime=${buildtime} -X ${pkg}.GitSHA=${sha} --extldflags '-static'"
}

mkdir -pv $builddir
rm -fv $builddir/*

build linux amd64
build windows amd64
build darwin amd64
build darwin arm64
