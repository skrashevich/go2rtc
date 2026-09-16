# Apple HomeKit

This module supports both client and server for the [Apple HomeKit](https://www.apple.com/home-app/accessories/) protocol.

## HomeKit Client

**Important:**

- You can use HomeKit Cameras **without Apple devices** (iPhone, iPad, etc.), it's just a yet another protocol
- HomeKit device can be paired with only one ecosystem. So, if you have paired it to an iPhone (Apple Home), you can't pair it with Home Assistant or go2rtc. Or if you have paired it to go2rtc, you can't pair it with an iPhone
- HomeKit device should be on the same network with working [mDNS](https://en.wikipedia.org/wiki/Multicast_DNS) between the device and go2rtc

go2rtc supports importing paired HomeKit devices from [Home Assistant](../hass/README.md). 
So you can use HomeKit camera with Home Assistant and go2rtc simultaneously. 
If you are using Home Assistant, I recommend pairing devices with it; it will give you more options.

You can pair device with go2rtc on the HomeKit page. If you can't see your devices, reload the page. 
Also, try rebooting your HomeKit device (power off). If you still can't see it, you have a problem with mDNS.

If you see a device but it does not have a pairing button, it is paired to some ecosystem (Apple Home, Home Assistant, HomeBridge, etc.). You need to delete the device from that ecosystem, and it will be available for pairing. If you cannot unpair the device, you will have to reset it.

**Important:**

- HomeKit audio uses very non-standard **AAC-ELD** codec with very non-standard params and specification violations
- Audio can't be played in `VLC` and probably any other player
- Audio should be transcoded for use with MSE, WebRTC, etc.

### Client Configuration

Recommended settings for using HomeKit Camera with WebRTC, MSE, MP4, RTSP:

```yaml
streams:
  aqara_g3:
    - hass:Camera-Hub-G3-AB12
    - ffmpeg:aqara_g3#audio=aac#audio=opus
```

RTSP link with "normal" audio for any player: `rtsp://192.168.1.123:8554/aqara_g3?video&audio=aac`

**This source is in active development!** Tested only with [Aqara Camera Hub G3](https://www.aqara.com/eu/product/camera-hub-g3) (both EU and CN versions).

## HomeKit Server

[`new in v1.7.0`](https://github.com/AlexxIT/go2rtc/releases/tag/v1.7.0)

HomeKit module can work in two modes:

- export any H264 camera to Apple HomeKit
- transparent proxy any Apple HomeKit camera (Aqara, Eve, Eufy, etc.) back to Apple HomeKit, so you will have all camera features in Apple Home and also will have RTSP/WebRTC/MP4/etc. from your HomeKit camera

**Important**

- HomeKit cameras support only H264 video and OPUS audio

### Server Configuration

**Minimal config**

```yaml
streams:
  dahua1: rtsp://admin:password@192.168.1.123/cam/realmonitor?channel=1&subtype=0
homekit:
  dahua1:  # same stream ID from streams list, default PIN - 19550224
```

**Full config**

```yaml
streams:
  dahua1:
    - rtsp://admin:password@192.168.1.123/cam/realmonitor?channel=1&subtype=0
    - ffmpeg:dahua1#video=h264#hardware  # if your camera doesn't support H264, important for HomeKit
    - ffmpeg:dahua1#audio=opus           # only OPUS audio supported by HomeKit

homekit:
  dahua1:                   # same stream ID from streams list
    pin: 12345678           # custom PIN, default: 19550224
    name: Dahua camera      # custom camera name, default: generated from stream ID
    device_id: dahua1       # custom ID, default: generated from stream ID
    device_private: dahua1  # custom key, default: generated from stream ID
    speaker: true           # enable 2-way audio (default: false, enable only if camera has a speaker)
```

### Camera controls in Apple Home

For cameras published by go2rtc (including `hksv: true`), Home's camera controls
now affect the running media pipeline:

- **Off** stops HomeKit live sessions and recordings, discards prepared recording
  media, suppresses motion events, and rejects new live/snapshot/recording requests.
- **Detect Activity** can report motion with live streaming disabled.
- **Stream** allows live video while recording remains disabled. Recording starts
  only after Home enables Recording Active.
- **Record Audio** determines whether new recording clips contain an AAC track.
  Changing it closes the current recording connection and discards prepared media;
  the home hub must reconnect before recording another clip with the new setting.
- Event snapshots (notifications) and periodic snapshots (preview tiles) obey their
  respective switches. Older requests without a reason are rejected if either
  class of snapshots is disabled.

These settings are written automatically to `homekit.<stream>.operating_state`
in `go2rtc.yaml` and restored on restart. Keep the configuration writable; a save
failure is logged and returned to Home as an error, although the runtime restriction
still takes effect. No re-pairing or new configuration flag is required. These
controls govern HomeKit output; other go2rtc clients and the source camera's own
recording are unaffected. An already displayed thumbnail may remain cached in Home.

To check after installing this branch: open a live view, switch the camera off,
verify that live view and fresh snapshots stop, then re-enable streaming. Check
that Stream-only creates no new clips, and toggle Record Audio before recording
another motion event. Finally restart go2rtc with the camera off and verify that
it stays off. Real Apple-device verification is still required.

### iOS 27 Secure Video development

The iOS 27 camera protocol is being implemented separately from classic HKSV.
The current foundation includes extended HAP type IDs, TLV8 fragmentation and
boolean support, synchronized event subscriptions, typed camera capabilities
and stream-tier descriptions, global operating-mode metadata and its streaming
gate, and motion-zone service metadata with version 2 encoding/decoding. Zone
validation checks sensor bounds, UUIDs, degenerate polygons and self-intersections.
It does **not** enable HEVC/4K HomeKit streaming or recording yet.

`mode: legacy` is optional and uses the classic protocol for `hksv: true`
and `hksv: false`, including the working controls described above. The future `mode: secure_video` is reserved: selecting it
currently logs an explicit error and skips publishing that camera. Unknown modes
are also rejected. Leave the mode unset for normal use.

The new capabilities service must not be attached to a classic camera: it tells
the home hub to use the new protocol. These service builders are not attached to
published cameras. The new service handlers still need admin-only and timed-write enforcement
and motion-zone processing. State persistence and media shutdown are implemented
for the classic services described above; the new service IDs remain unpublished. The next media steps are multi-tier HEVC RTP, HEVC recording
over HDS (including fragment timestamps), then WebRTC/SFrame and remote audio.
CMAF direct upload is a separate follow-up. No automatic mode selection or
migration of existing pairings is performed.

Protocol reference: [Apple's HKSV Open Source Compatibility Guide](https://developer.apple.com/download/files/HomeKit-Secure-Video-Open-Source-Compatibility-Guide.pdf)
(Developer Preview, June 3, 2026). Wire definitions are in
`pkg/hap/camera/secure_video.go`; metadata is provided by callers, not inferred
from a camera name or a fixed list of advertised resolutions.

Foundation checks:

```sh
go test -race ./pkg/hap/... ./pkg/homekit/... ./pkg/hksv/... ./internal/homekit/...
```

These checks cover encoding and server behavior. They do not substitute for
pairing, streaming, recording, and privacy tests on real iOS/tvOS 27 devices.

### HKSV (HomeKit Secure Video)

go2rtc can expose any camera as a HomeKit Secure Video (HKSV) camera. This allows Apple Home to record video clips to iCloud when motion is detected.

**Requirements:**
- Apple Home Hub (Apple TV, HomePod or iPad) on the same network
- iCloud storage plan with HomeKit Secure Video support
- Camera source with H264 video (AAC audio recommended)

**Minimal HKSV config**

```yaml
streams:
  outdoor: rtsp://admin:password@192.168.1.123/stream1

homekit:
  outdoor:
    hksv: true           # enable HomeKit Secure Video
    motion: continuous   # always report motion, Home Hub decides what to record
```

**Full HKSV config**

```yaml
streams:
  outdoor:
    - rtsp://admin:password@192.168.1.123/stream1
    - ffmpeg:outdoor#video=h264#hardware  # transcode to H264 if needed
    - ffmpeg:outdoor#audio=aac            # AAC-LC audio for HKSV recording

homekit:
  outdoor:
    pin: 12345678
    name: Outdoor Camera
    hksv: true
    motion: api          # motion triggered via API
```

**HKSV Doorbell config**

```yaml
homekit:
  front_door:
    category_id: doorbell
    hksv: true
    motion: api
```

**Motion modes:**

- `continuous` — MotionDetected is always true; Home Hub continuously receives video and decides what to save. Simplest setup, recommended for most cameras.
- `detect` — automatic motion detection by analyzing H264 P-frame sizes. No external dependencies or CPU-heavy decoding. Works with any H264 source and resolution. Compares each P-frame size against an adaptive baseline using EMA (exponential moving average). When a P-frame exceeds the threshold ratio, motion is triggered with a 30s hold time and 5s cooldown.
- `api` — motion is triggered externally via HTTP API. Use this with Frigate, ONVIF events, or any other motion detection system.

**Motion detect config:**

```yaml
homekit:
  outdoor:
    hksv: true
    motion: detect
    motion_threshold: 1.0  # P-frame size / baseline ratio to trigger motion (default: 2.0)
```

The `motion_threshold` controls sensitivity — it's the ratio of P-frame size to the adaptive baseline. When a P-frame exceeds `baseline × threshold`, motion is triggered.

| Scenario | threshold | Notes |
|---|---|---|
| Quiet indoor scene | 1.3–1.5 | Low noise, stable baseline, even small motion is visible |
| Standard camera (yard, hallway) | 2.0 (default) | Good balance between sensitivity and false positives |
| Outdoor with trees/shadows/wind | 2.5–3.0 | Wind and shadows produce medium P-frames, need margin |
| Busy street / complex scene | 3.0–5.0 | Lots of background motion, react only to large events |

Values below 1.0 are meaningless (triggers on every frame). Values above 5.0 require very large motion (person filling half the frame).

**How to tune:** set `log.level: trace` and watch `motion: status` lines — they show current `ratio`. Walk in front of the camera and note the ratio values:

```
motion: status baseline=5000 ratio=0.95  ← quiet
motion: status baseline=5000 ratio=3.21  ← person walked by
motion: status baseline=5000 ratio=1.40  ← shadow/wind
```

Set threshold between "noise" and "real motion". In this example, 2.0 is a good choice (ignores 1.4, catches 3.2).

**Motion API:**

```bash
# Get motion status
curl "http://localhost:1984/api/homekit/motion?id=outdoor"
# → {"id":"outdoor","motion":false}

# Trigger motion start
curl -X POST "http://localhost:1984/api/homekit/motion?id=outdoor"

# Clear motion
curl -X DELETE "http://localhost:1984/api/homekit/motion?id=outdoor"

# Trigger doorbell ring
curl -X POST "http://localhost:1984/api/homekit/doorbell?id=front_door"
```

**Proxy HomeKit camera**

- Video stream from HomeKit camera to Apple device (iPhone, Apple TV) will be transmitted directly
- Video stream from HomeKit camera to RTSP/WebRTC/MP4/etc. will be transmitted via go2rtc

```yaml
streams:
  aqara1:
    - homekit://...
    - ffmpeg:aqara1#audio=aac#audio=opus  # optional audio transcoding

homekit:
  aqara1:  # same stream ID from streams list
```
