# Use of this source code is governed by Apache License 2.0
# that can be found in the LICENSE file.

FROM golang:1.19-alpine3.16 AS build-stage

COPY . /opt/src
WORKDIR /opt/src

ENV GO111MODULE on

# prepare for build
RUN apk add --no-cache build-base git
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

COPY --from=build-stage /opt/src/flowgre ./flowgre

# override default entrypoint on final container
ENTRYPOINT [ "/opt/app/flowgre" ]
LABEL org.opencontainers.image.source=https://github.com/dmabry/flowgre
LABEL org.opencontainers.image.description="Flowgre container image"
LABEL org.opencontainers.image.licenses="Apache License 2.0"
LABEL org.opencontainers.image.version="0.4.0"
