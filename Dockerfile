# syntax=docker/dockerfile:labs

# 0. Prepare images
ARG PYTHON_VERSION="3.11"
ARG GO_VERSION="1.20"
ARG NGROK_VERSION="3"

FROM python:${PYTHON_VERSION}-alpine AS base
# Install ffmpeg, tini (for signal handling),
# and other common tools for the echo source.
RUN apk add --no-cache tini ffmpeg bash curl jq

# Hardware Acceleration for Intel CPU (+50MB)
ARG TARGETARCH

RUN if [ "${TARGETARCH}" = "amd64" ]; then apk add --no-cache libva-intel-driver intel-media-driver; fi

FROM ngrok/ngrok:${NGROK_VERSION}-alpine AS ngrok


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
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -pgo=./cpu.pprof

# 2. Final image
FROM base

# Hardware: AMD and NVidia VAAPI (not sure about this)
# RUN libva-glx mesa-va-gallium
# Hardware: AMD and NVidia VDPAU (not sure about this)
# RUN libva-vdpau-driver mesa-vdpau-gallium (+150MB total)

COPY --link --from=build /build/go2rtc /usr/local/bin/
COPY --link --from=ngrok /bin/ngrok /usr/local/bin/

ENTRYPOINT ["/sbin/tini", "--"]
VOLUME /config
WORKDIR /config

CMD ["go2rtc", "-config", "/config/go2rtc.yaml"]
