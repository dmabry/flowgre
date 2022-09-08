# Use of this source code is governed by Apache License 2.0
# that can be found in the LICENSE file.

FROM golang:1.19-alpine3.16 AS build-stage

COPY . /opt/src
WORKDIR /opt/src

ENV GO111MODULE on

# build
RUN git submodule -q init
RUN git submodule -q update
# RUN go build -mod vendor
RUN go build

# deploy
FROM alpine:3.16

# add some alpine deps
RUN apk add --no-cache tzdata

# copy stuff in
WORKDIR /opt/app

COPY --from=build-stage /opt/src/flowgre	./flowgre

# override default entrypoint on final container
ENTRYPOINT [ "/opt/app/flowgre" ]
