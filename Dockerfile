# Use of this source code is governed by Apache License 2.0
# that can be found in the LICENSE file.

# ---------------------------------------------------------------------------
# Build stage — compile a fully static binary (CGO_ENABLED=0)
# ---------------------------------------------------------------------------
FROM golang:1.26-alpine3.24 AS build-stage

WORKDIR /src

# Cache dependency resolution layer separately from source code.
COPY go.mod go.sum ./
RUN go mod download

ARG VERSION=dev

# Copy source and build a static binary.
# TARGETARCH is auto-injected by BuildKit (arm64 on this host, amd64 on CI).
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags "-s -w -X main.version=${VERSION}" -o /flowgre .

# ---------------------------------------------------------------------------
# Runtime stage — minimal Alpine image, non-root user
# ---------------------------------------------------------------------------
FROM alpine:3.24

RUN addgroup -S flowgre && \
    adduser -S -g flowgre flowgre

WORKDIR /opt/app
COPY --chown=flowgre:flowgre --from=build-stage /flowgre ./flowgre

USER flowgre

ENTRYPOINT ["/opt/app/flowgre"]

# OCI labels — version is injected at build time via --build-arg
ARG VERSION=dev
LABEL org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.source="https://github.com/dmabry/flowgre" \
      org.opencontainers.image.description="NetFlow v9 / IPFIX packet generator for collector stress testing" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.vendor="Flowgre Team"

# No HEALTHCHECK — flowgre is a CLI tool, not a daemon.
# The /health endpoint only exists in barrage mode with -web flag.
