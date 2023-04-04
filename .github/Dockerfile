FROM node:18 AS frontend_build

COPY frontend /app

RUN set -xe; \
    cd /app; \
    yarn install; \
    yarn build; \
    mv dist webui;

# Need to build with earliest possible Ubuntu version to ensure the lowest
# possible libc version we can reasonably link to to ensure the output binaries
# will work on all platforms.
FROM ubuntu:18.04

ARG GO_VERSION=1.20.3

RUN set -xe; \
    export DEBIAN_FRONTEND=noninteractive; \
    apt-get update; \
    apt-get install -y wget xz-utils git make; \
    cd /tmp; \
    mkdir -p /opt/go; \
    wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz; \
    tar -C /opt/go --strip-components=1 -xf go${GO_VERSION}.linux-amd64.tar.gz; \
    rm /tmp/*

COPY . /app
COPY --from=frontend_build /app/webui /app/webui

RUN set -xe; \
    cd /app; \
    export PATH=/opt/go/bin:$PATH; \
    # Necessary for dirty tree detection
    git update-index -q --really-refresh; \
    make prod-all; \
