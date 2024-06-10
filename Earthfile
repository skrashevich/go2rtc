# syntax = docker+earthly:latest

VERSION 0.8

# Base stage to check dependencies
check-deps:
    FROM alpine:3.19
    RUN apk add --no-cache go upx p7zip
    RUN which go && which 7z && which upx

# Base stage to build for all platforms
build:
    FROM golang:1.22-alpine
    WORKDIR /src
    COPY . .
    RUN apk add --no-cache upx p7zip

# Build Windows amd64
build-windows-amd64:
    FROM +build AS build-windows
    ARG GOOS=windows
    ARG GOARCH=amd64
    RUN go build -ldflags "-s -w" -trimpath -o /output/go2rtc.exe .
    RUN 7z a -mx9 -bso0 -sdel /output/go2rtc_win64.zip /output/go2rtc.exe
    SAVE ARTIFACT /output/go2rtc_win64.zip AS LOCAL go2rtc_win64.zip

# Build Windows 386
build-windows-386:
    FROM +build AS build-windows
    ARG GOOS=windows
    ARG GOARCH=386
    RUN go build -ldflags "-s -w" -trimpath -o /output/go2rtc.exe .
    RUN 7z a -mx9 -bso0 -sdel /output/go2rtc_win32.zip /output/go2rtc.exe
    SAVE ARTIFACT /output/go2rtc_win32.zip AS LOCAL go2rtc_win32.zip

# Build Windows arm64
build-windows-arm64:
    FROM +build AS build-windows
    ARG GOOS=windows
    ARG GOARCH=arm64
    RUN go build -ldflags "-s -w" -trimpath -o /output/go2rtc.exe .
    RUN 7z a -mx9 -bso0 -sdel /output/go2rtc_win_arm64.zip /output/go2rtc.exe
    SAVE ARTIFACT /output/go2rtc_win_arm64.zip AS LOCAL go2rtc_win_arm64.zip

# Build Linux amd64
build-linux-amd64:
    FROM +build AS build-linux
    ARG GOOS=linux
    ARG GOARCH=amd64
    RUN go build -ldflags "-s -w" -trimpath -o /output/go2rtc_linux_amd64 .
    RUN upx --lzma --force-overwrite -q --no-progress /output/go2rtc_linux_amd64
    SAVE ARTIFACT /output/go2rtc_linux_amd64 AS LOCAL go2rtc_linux_amd64

# Build Linux 386
build-linux-386:
    FROM +build AS build-linux
    ARG GOOS=linux
    ARG GOARCH=386
    RUN go build -ldflags "-s -w" -trimpath -o /output/go2rtc_linux_i386 .
    RUN upx --lzma --force-overwrite -q --no-progress /output/go2rtc_linux_i386
    SAVE ARTIFACT /output/go2rtc_linux_i386 AS LOCAL go2rtc_linux_i386

# Build Linux arm64
build-linux-arm64:
    FROM +build AS build-linux
    ARG GOOS=linux
    ARG GOARCH=arm64
    RUN go build -ldflags "-s -w" -trimpath -o /output/go2rtc_linux_arm64 .
    RUN upx --lzma --force-overwrite -q --no-progress /output/go2rtc_linux_arm64
    SAVE ARTIFACT /output/go2rtc_linux_arm64 AS LOCAL go2rtc_linux_arm64

# Build Linux arm v7
build-linux-armv7:
    FROM +build AS build-linux
    ARG GOOS=linux
    ARG GOARCH=arm
    ARG GOARM=7
    RUN go build -ldflags "-s -w" -trimpath -o /output/go2rtc_linux_arm .
    RUN upx --lzma --force-overwrite -q --no-progress /output/go2rtc_linux_arm
    SAVE ARTIFACT /output/go2rtc_linux_arm AS LOCAL go2rtc_linux_arm

# Build Linux arm v6
build-linux-armv6:
    FROM +build AS build-linux
    ARG GOOS=linux
    ARG GOARCH=arm
    ARG GOARM=6
    RUN go build -ldflags "-s -w" -trimpath -o /output/go2rtc_linux_armv6 .
    RUN upx --lzma --force-overwrite -q --no-progress /output/go2rtc_linux_armv6
    SAVE ARTIFACT /output/go2rtc_linux_armv6 AS LOCAL go2rtc_linux_armv6

# Build Linux mipsle
build-linux-mipsle:
    FROM +build AS build-linux
    ARG GOOS=linux
    ARG GOARCH=mipsle
    RUN go build -ldflags "-s -w" -trimpath -o /output/go2rtc_linux_mipsel .
    RUN upx --lzma --force-overwrite -q --no-progress /output/go2rtc_linux_mipsel
    SAVE ARTIFACT /output/go2rtc_linux_mipsel AS LOCAL go2rtc_linux_mipsel

# Build Darwin amd64
build-darwin-amd64:
    FROM +build AS build-darwin
    ARG GOOS=darwin
    ARG GOARCH=amd64
    RUN go build -ldflags "-s -w" -trimpath -o /output/go2rtc .
    RUN 7z a -mx9 -bso0 -sdel /output/go2rtc_mac_amd64.zip /output/go2rtc
    SAVE ARTIFACT /output/go2rtc_mac_amd64.zip AS LOCAL go2rtc_mac_amd64.zip

# Build Darwin arm64
build-darwin-arm64:
    FROM +build
    ARG GOOS=darwin
    ARG GOARCH=arm64
    RUN go build -ldflags "-s -w" -trimpath -o /output/go2rtc .
    RUN 7z a -mx9 -bso0 -sdel /output/go2rtc_mac_arm64.zip /output/go2rtc
    SAVE ARTIFACT /output/go2rtc_mac_arm64.zip AS LOCAL go2rtc_mac_arm64.zip
