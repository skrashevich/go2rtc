This examples for Home Assistant [Telegram Bot](https://www.home-assistant.io/integrations/telegram_bot/) integration.

- change `url` to your go2rtc web API (`http://localhost:1984/` for most users)
- change `target` to your Telegram chat ID (support list)
- change `src=camera1` to your stream name from go2rtc config

**Important.** Snapshot will be near instant for most cameras and many sources, except `ffmpeg` source. Because it takes a long time for ffmpeg to start streaming with video, even when you use `#video=copy`. Also the delay can be with cameras that do not start the stream with a keyframe.

## Snapshot from H264 or H265 camera

```yaml
service: telegram_bot.send_video
data:
  url: http://localhost:1984/api/frame.mp4?src=camera1
  target: 123456789
```

## Snapshot from JPEG or MJPEG camera

```yaml
service: telegram_bot.send_photo
data:
  url: http://localhost:1984/api/frame.jpeg?src=camera1
  target: 123456789
```

## Record from H264 or H265 camera

Record from service call to the future. Doesn't support loopback.

- `mp4=flac` - adds support PCM audio family
- `filename=record.mp4` - set name for downloaded file

```yaml
service: telegram_bot.send_video
data:
  url: http://localhost:1984/api/stream.mp4?src=camera1&mp4=flac&duration=5&filename=record.mp4  # duration in seconds
  target: 123456789
```
