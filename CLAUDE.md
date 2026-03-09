# CLAUDE.md — go2rtc

## Project Overview

go2rtc is an ultimate camera streaming application written in Go. It supports dozens of streaming formats and protocols (RTSP, WebRTC, RTMP, HLS, HomeKit, and 40+ camera-specific integrations) and produces zero-dependency small binaries for all major operating systems.

- **Language**: Go 1.24+
- **Module path**: `github.com/AlexxIT/go2rtc`
- **Entry point**: `main.go`
- **Current version**: 1.9.14

## Repository Structure

```
main.go              # Entry point — registers and initializes all modules
go.mod / go.sum      # Go module dependencies
internal/            # Application modules (46 packages) — NOT importable externally
  app/               # Config loading, logging, module initialization
  api/               # HTTP API server, auth, CORS, TLS
  api/ws/            # WebSocket API endpoint
  streams/           # Stream management and registry
  rtsp/              # RTSP client/server
  webrtc/            # WebRTC client/server
  ffmpeg/            # FFmpeg integration
  homekit/           # Apple HomeKit server
  ...                # 35+ protocol/device-specific modules
pkg/                 # Reusable libraries (66 packages) — importable by external projects
  core/              # Core interfaces: Producer, Consumer, Media, Codec, Receiver, Sender
  h264/, h265/       # Video codec parsers
  aac/, opus/        # Audio codec parsers
  mp4/, mjpeg/       # Container format handlers
  rtsp/, rtmp/       # Protocol implementations
  shell/             # Signal handling, process utilities
  yaml/              # Config parsing helpers
  ...
www/                 # Web UI static files
website/             # Documentation site (VitePress)
docker/              # Dockerfiles (standard, hardware, rockchip)
scripts/             # Build scripts
.github/workflows/   # CI/CD (build.yml, test.yml, gh-pages.yml)
```

## Architecture

### Core Concepts

- **Producer**: Generates media streams (implements `pkg/core.Producer` — `GetMedias()`, `GetTrack()`, `Start()`, `Stop()`)
- **Consumer**: Receives media streams (implements `pkg/core.Consumer` — `GetMedias()`, `AddTrack()`, `Stop()`)
- **Media**: Describes a media stream (video/audio) with direction and supported codecs
- **Codec**: Represents a specific codec (H264, H265, PCMA, Opus, etc.)
- **Receiver/Sender**: Handle RTP packet flow between producers and consumers

### Module System

Each feature is an `internal/` package with an `Init()` function. Modules are registered in `main.go` in a specific order and selectively initialized based on config:

1. **Core infrastructure**: app, api, ws, streams
2. **Main servers**: http, rtsp, webrtc
3. **Output formats**: mp4, hls, mjpeg
4. **Specialized servers**: hass, homekit, onvif, rtmp, webtorrent, wyoming
5. **Script sources**: echo, exec, expr, ffmpeg
6. **Hardware sources**: alsa, v4l2
7. **Device integrations**: bubble, doorbird, dvrip, kasa, tapo, xiaomi, etc.
8. **Helpers**: debug, ngrok, pinggy, srtp

### Configuration

YAML-based (`go2rtc.yaml`). Each module reads its own config section in `Init()`. Default ports:
- HTTP API: `:1984`
- RTSP: `:8554`
- WebRTC: `:8555/tcp` and `:8555/udp`

## Build & Run

### Building

```bash
# Standard build (CGO disabled, no native dependencies)
CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath

# Run directly
go run .
```

There is no Makefile. The project uses Go's native build system.

### Docker

Three variants in `docker/`:
- `Dockerfile` — Alpine + Python + FFmpeg
- `hardware.Dockerfile` — adds Intel GPU drivers
- `rockchip.Dockerfile` — Rockchip device support

### Running Tests

```bash
go test ./...
```

Tests use `github.com/stretchr/testify/require` for assertions. Test files live alongside source code (`*_test.go` in the same package).

## Code Conventions

### Package Organization

- `pkg/{format}/producer.go` — producer for a format
- `pkg/{format}/consumer.go` — consumer for a format
- `pkg/{format}/backchannel.go` — producer with backchannel (two-way audio) only

### Naming

go2rtc names formats, protocols, and codecs the same way FFmpeg does. Use FFmpeg naming conventions when adding new codec/format support.

### Imports

Standard library → external dependencies → internal packages (goimports ordering).

### Logging

Uses `github.com/rs/zerolog` with per-module loggers. Each module creates its own logger.

### Error Handling

Named error variables (e.g., `ErrCantGetTrack`). Errors are propagated with context. Minimal custom error types.

### Testing Patterns

- Unit tests in the same package as source code
- Mock implementations of `Producer`/`Consumer` interfaces for testing
- Use `require` (not `assert`) from testify for test assertions
- No global test fixtures

## Adding a New Module

When adding a new module, update these files:

1. `main.go` — add import and register in the modules slice
2. `README.md` — document the new source/feature
3. `internal/README.md` — add to the modules table
4. `website/.vitepress/config.js` — add to documentation navigation
5. `website/api/openapi.yaml` — if it exposes API endpoints
6. `www/schema.json` — if it has configuration options

## CI/CD

- **build.yml**: Cross-compiles for 12 platform/arch combinations, builds Docker images, pushes to GHCR
- **test.yml**: Tests on Windows, macOS, Linux (amd64 + arm64 via QEMU)
- **gh-pages.yml**: Builds and publishes documentation site

Build flags: `CGO_ENABLED=0`, `-ldflags "-s -w"`, `-trimpath`

## Key Dependencies

- `github.com/pion/webrtc` — WebRTC stack
- `github.com/rs/zerolog` — Structured logging
- `github.com/stretchr/testify` — Test assertions
- `github.com/brutella/hap` — HomeKit Accessory Protocol
- No CGO dependencies — all binaries are statically linked
