# syntax=docker/dockerfile:labs

# 0. Prepare images
# only debian 12 (bookworm) has latest ffmpeg
ARG DEBIAN_VERSION="bookworm-slim"
ARG GO_VERSION="1.21-bookworm"
ARG NGROK_VERSION="3"

FROM debian:${DEBIAN_VERSION} AS base
FROM golang:${GO_VERSION} AS go
FROM ngrok/ngrok:${NGROK_VERSION} AS ngrok


# 1. Build go2rtc binary
FROM --platform=$BUILDPLATFORM go AS build
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
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath


# 2. Collect all files
FROM scratch AS rootfs

COPY --link --from=build /build/go2rtc /usr/local/bin/
COPY --link --from=ngrok /bin/ngrok /usr/local/bin/

# 3. Final image
FROM base
ARG TARGETARCH
# Prepare apt for buildkit cache
RUN rm -f /etc/apt/apt.conf.d/docker-clean \
  && echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' >/etc/apt/apt.conf.d/keep-cache
# Install ffmpeg, bash (for run.sh), tini (for signal handling),
# and other common tools for the echo source.
# non-free for Intel QSV support (not used by go2rtc, just for tests)
# libasound2-plugins for ALSA support
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked --mount=type=cache,target=/var/lib/apt,sharing=locked \
    echo 'deb http://deb.debian.org/debian bookworm non-free' > /etc/apt/sources.list.d/debian-non-free.list && \
    apt-get -y update && apt-get -y install tini \
        python3 curl jq \
        intel-media-va-driver-non-free \
        libasound2-plugins
RUN <<EOT
  # btbn-ffmpeg -> amd64
  if [[ "${TARGETARCH}" == "amd64" ]]; then
      mkdir -p /usr/lib/btbn-ffmpeg
      wget -qO btbn-ffmpeg.tar.xz "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-n6.0-latest-linux64-gpl-6.0.tar.xz"
      tar -xf btbn-ffmpeg.tar.xz -C /usr/lib/btbn-ffmpeg --strip-components 1
      rm -rf btbn-ffmpeg.tar.xz /usr/lib/btbn-ffmpeg/doc /usr/lib/btbn-ffmpeg/bin/ffplay
  fi
EOT

COPY --link --from=rootfs / /


ENV PATH="/usr/lib/btbn-ffmpeg/bin:${PATH}"

ENTRYPOINT ["/usr/bin/tini", "--"]
VOLUME /config
WORKDIR /config
# https://github.com/NVIDIA/nvidia-docker/wiki/Installation-(Native-GPU-Support)
ENV NVIDIA_VISIBLE_DEVICES all
ENV NVIDIA_DRIVER_CAPABILITIES compute,video,utility

CMD ["go2rtc", "-config", "/config/go2rtc.yaml"]
