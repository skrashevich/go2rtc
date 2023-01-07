# syntax=docker/dockerfile:labs

# 0. Prepare images
ARG PYTHON_VERSION="3.11"
ARG GO_VERSION="1.19"
ARG NGROK_VERSION="3"

FROM python:${PYTHON_VERSION}-slim AS base
FROM ngrok/ngrok:${NGROK_VERSION}-alpine AS ngrok

# 0. collect ace editor
FROM alpine:latest as ace
RUN apk add curl
RUN <<EOT
    for i in \
        https://cdn.jsdelivr.net/npm/ace-builds@1.14.0/src-min-noconflict/ace.min.js \
        https://cdn.jsdelivr.net/npm/ace-builds@1.14.0/src-min-noconflict/mode-yaml.min.js \
        https://cdn.jsdelivr.net/npm/ace-builds@1.14.0/src-min-noconflict/worker-yaml.min.js \
        https://cdn.jsdelivr.net/npm/ace-builds@1.14.0/src-min-noconflict/theme-terminal.min.js \
        https://cdn.jsdelivr.net/npm/ace-builds@1.14.0/src-min-noconflict/theme-monokai.min.js
    do
        curl -sLk "$i" >> /ace.js; echo "" >> /ace.js;
    done
EOT

# 1. Build go2rtc binary
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build go mod download

COPY . .
COPY --from=ace /ace.js www/ace.js
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath

# 2. Collect all files
FROM scratch AS rootfs

COPY --from=build /build/go2rtc /usr/local/bin/
COPY --from=ngrok /bin/ngrok /usr/local/bin/
COPY ./build/docker/run.sh /


# 3. Final image
FROM base
ARG TARGETARCH
# Install ffmpeg, bash (for run.sh), tini (for signal handling),
# and other common tools for the echo source.
RUN apt update && apt-get install -y --no-upgrade  bash tini curl jq xz-utils && apt-get clean

# Download and install ffmpeg binaries
RUN <<EOT
    #!/bin/bash

    # btbn-ffmpeg -> amd64 / arm64
    if [[ "${TARGETARCH}" == "amd64" || "${TARGETARCH}" == "arm64" ]]; then
        if [[ "${TARGETARCH}" == "amd64" ]]; then
            btbn_arch="64"
        else
            btbn_arch="arm64"
        fi
        mkdir -p /usr/lib/btbn-ffmpeg
        curl -sL -o btbn-ffmpeg.tar.xz "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-n5.1-latest-linux${btbn_arch}-gpl-shared-5.1.tar.xz"
        tar -Jxf btbn-ffmpeg.tar.xz -C /usr/lib/btbn-ffmpeg --strip-components 1
        rm -rf btbn-ffmpeg.tar.xz /usr/lib/btbn-ffmpeg/{doc,include}
    fi

    # ffmpeg -> arm32
    if [[ "${TARGETARCH}" == "arm" ]]; then
    apt update && apt install -y ffmpeg
    fi
EOT

ENV NVIDIA_DRIVER_CAPABILITIES="compute,video,utility"

ENV PATH="/usr/lib/btbn-ffmpeg/bin:${PATH}"

COPY --from=rootfs / /

ENTRYPOINT ["/usr/bin/tini", "--"]

CMD ["/run.sh"]
