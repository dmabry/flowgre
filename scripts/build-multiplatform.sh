#!/bin/bash
oses=(windows darwin linux)
archs=(amd64 arm64)

for os in ${oses[@]}
do
  for arch in ${archs[@]}
  do
        env GOOS=${os} GOARCH=${arch} go build -o flowgre_${os}_${arch}
  done
done
