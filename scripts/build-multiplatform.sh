#!/bin/bash
# Use of this source code is governed by Apache License 2.0
# that can be found in the LICENSE file.

# Read first argument and pass it as tag otherwise set test
[ -z "$1" ] && tag="test" || tag=$1

oses=(windows darwin linux)
archs=(amd64 arm64)

for os in ${oses[@]}
do
  for arch in ${archs[@]}
  do
	  echo "Building binary flowgre_${os}_${arch}_${tag}"
    env GOOS=${os} GOARCH=${arch} go build -ldflags '-s -w' -a -o flowgre_${os}_${arch}_${tag}
  done
done
