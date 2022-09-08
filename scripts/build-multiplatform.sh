#!/bin/bash
# Use of this source code is governed by Apache License 2.0
# that can be found in the LICENSE file.

oses=(windows darwin linux)
archs=(amd64 arm64)

for os in ${oses[@]}
do
  for arch in ${archs[@]}
  do
        env GOOS=${os} GOARCH=${arch} go build -o flowgre_${os}_${arch}
  done
done
