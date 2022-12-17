# https://hub.docker.com/_/python/tags?page=1&name=-alpine
ARG PYTHON_VERSION="3.10.8"
# https://hub.docker.com/_/golang/tags?page=1&name=-alpine
ARG GO_VERSION="1.19.3"
# https://hub.docker.com/r/ngrok/ngrok/tags?page=1&name=-alpine
ARG NGROK_VERSION="3.1.0"


FROM python:${PYTHON_VERSION}-slim AS base


FROM golang:${GO_VERSION} AS go


FROM ngrok/ngrok:${NGROK_VERSION}-alpine AS ngrok


# Build go2rtc binary
FROM go AS build

WORKDIR /workspace

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build binary
COPY cmd cmd
COPY pkg pkg
COPY www www
COPY main.go .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath

RUN mkdir -p /config

# Collect all files
FROM scratch AS rootfs

COPY --from=build /workspace/go2rtc /usr/local/bin/
# Ensure an empty /config folder exists so that the container can be run without a volume
COPY --from=build /config /config
COPY --from=ngrok /bin/ngrok /usr/local/bin/
COPY ./docker/run.sh /run.sh


# Final image
FROM base
ARG TARGETARCH
# Install ffmpeg, bash (for run.sh), tini (for signal handling),
# and other common tools for the echo source.
RUN apt update && apt-get install -y --no-upgrade  bash tini curl jq xz-utils && apt-get clean

# Download and install ffmpeg binaries
RUN --mount=type=bind,source=docker/ffmpeg.sh,target=/deps/ffmpeg.sh \
        /deps/ffmpeg.sh

ENV NVIDIA_DRIVER_CAPABILITIES="compute,video,utility"

ENV PATH="/usr/lib/btbn-ffmpeg/bin:${PATH}"

COPY --from=rootfs / /

ENTRYPOINT ["/usr/bin/tini", "--"]

CMD ["/run.sh"]
