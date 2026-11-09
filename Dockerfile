# syntax = docker/dockerfile:1.4

FROM alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a as RUN

# Add runtime dependencies
# - tzdata: Go time required external dependency eg: TRANSIP and possibly others
# - ca-certificates: Needed for https to work properly
RUN apk update && apk add --no-cache tzdata ca-certificates && update-ca-certificates

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/dnscontrol /usr/local/bin/

WORKDIR /dns

ENTRYPOINT ["/usr/local/bin/dnscontrol"]
