## Useful links

- [RFC 3550: RTP: A Transport Protocol for Real-Time Applications](https://datatracker.ietf.org/doc/html/rfc3550)
- [RFC 6716: Definition of the Opus Audio Codec](https://datatracker.ietf.org/doc/html/rfc6716)
- [RFC 7587: RTP Payload Format for the Opus Speech and Audio Codec](https://datatracker.ietf.org/doc/html/rfc7587)

## Built-in audio conversion

When there is no directly compatible audio track, go2rtc can convert in either direction:

| Source | Destination | Typical use |
| --- | --- | --- |
| `PCMA`, `PCMU`, `PCM`, `PCML` | Opus | WebRTC playback, HomeKit audio, RTSP/WebRTC clients requesting Opus, and PCM sources such as Wyoming |
| Opus | `PCMA`, `PCMU`, `PCM`, `PCML` | Camera speaker backchannels, playback to G.711 devices, and linear audio consumers |
| Opus | `PCML` then FLAC | MSE/MP4/HLS when a client requests FLAC, for example `?mp4=flac` |

Direct codec matches are preferred. Opus is sent with a 48 kHz RTP clock and 20 ms packets. The encoder resamples PCM to a supported Opus rate, enables in-band FEC with a 10% expected packet loss setting, and the decoder attempts FEC or packet loss concealment for short RTP gaps. Encoder and decoder state is released when the converted track detaches.

The conversion does not add AAC, MP3, or video transcoding; use FFmpeg for those codecs. It is unavailable on `linux/mipsle`, where `go-opus`'s `modernc.org/libc` dependency cannot build. Direct Opus passthrough continues to work there.

## Multichannel status

`go-opus` now tests multistream Opus output for four-channel and 5.1 audio. The exported encoder/decoder API used here still handles one or two channels. Multichannel support in go2rtc would also need exported multistream entry points, a channel mapping, and a container or protocol path that carries that mapping. Current WebRTC, HomeKit, and built-in Opus conversion remain mono/stereo; multichannel packets must not be sent through the single-stream path. See [RFC 7845 channel mapping](https://www.rfc-editor.org/rfc/rfc7845.html#section-5.1.1).
