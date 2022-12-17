# 0. Prepare images
ARG PYTHON_VERSION="3.11"
ARG GO_VERSION="1.19"
ARG NGROK_VERSION="3"

FROM python:${PYTHON_VERSION}-slim AS base


FROM golang:${GO_VERSION} AS go


FROM ngrok/ngrok:${NGROK_VERSION}-alpine AS ngrok


# 1. Build go2rtc binary
FROM go AS build

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath


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
RUN --mount=type=bind,source=docker/ffmpeg.sh,target=/deps/ffmpeg.sh \
        /deps/ffmpeg.sh

ENV NVIDIA_DRIVER_CAPABILITIES="compute,video,utility"

ENV PATH="/usr/lib/btbn-ffmpeg/bin:${PATH}"

COPY --from=rootfs / /

ENTRYPOINT ["/usr/bin/tini", "--"]

CMD ["/run.sh"]
