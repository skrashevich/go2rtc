#!/bin/bash

set -euxo pipefail

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

