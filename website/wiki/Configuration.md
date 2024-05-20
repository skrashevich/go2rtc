- By default go2rtc will search config file `go2rtc.yaml` in current work directory
- go2rtc support multiple config files
- go2rtc support config as raw YAML format from command line
- Every next config will overwrite previous (but only defined params)

```
go2rtc -config "{log: {format: text}}" -config /config/go2rtc.yaml -config "{rtsp: {listen: ''}}" -config /usr/local/go2rtc/go2rtc.yaml
```

## Environment variables

Also go2rtc support templates for using environment variables in any part of config:

```yaml
streams:
  camera1: rtsp://rtsp:${CAMERA_PASSWORD}@192.168.1.123/av_stream/ch0

${LOGS:}  # empty default value

rtsp:
  username: ${RTSP_USER:admin}   # "admin" if env "RTSP_USER" not set
  password: ${RTSP_PASS:secret}  # "secret" if env "RTSP_PASS" not set
```

## Defaults

```yaml
api:
  listen: ":1984"
  base_path: ""
  static_dir: ""
  origin: ""

ffmpeg:
  bin: "ffmpeg"
  global: "-hide_banner"
  file: "-re -stream_loop -1 -i {input}"
  http: "-fflags nobuffer -flags low_delay -i {input}"
  rtsp: "-fflags nobuffer -flags low_delay -timeout 5000000 -user_agent go2rtc/ffmpeg -rtsp_transport tcp -i {input}"
  output: "-user_agent ffmpeg/go2rtc -rtsp_transport tcp -f rtsp {output}"
  # ... different presets for codecs

hass:
  config: ""

log:
  format: ""
  level: "info"

ngrok:
  command: ""

rtsp:
  listen: ":8554"
  username: ""
  password: ""

srtp:
  listen: ":8443"

streams: {}

webrtc:
  listen: ":8555"
  candidates: []
  ice_servers:
    - urls: [ "stun:stun.l.google.com:19302" ]
      username: ""
      credential: ""
```