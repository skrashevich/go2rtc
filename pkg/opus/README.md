## Useful links

- [RFC 3550: RTP: A Transport Protocol for Real-Time Applications](https://datatracker.ietf.org/doc/html/rfc3550)
- [RFC 6716: Definition of the Opus Audio Codec](https://datatracker.ietf.org/doc/html/rfc6716)
- [RFC 7587: RTP Payload Format for the Opus Speech and Audio Codec](https://datatracker.ietf.org/doc/html/rfc7587)

## Built-in audio conversion

Participating consumers advertise additional audio input formats and convert them locally:

| Source | Destination | Typical use |
| --- | --- | --- |
| `PCMA`, `PCMU`, `PCM`, `PCML` | Opus | WebRTC playback, HomeKit audio, RTSP/WebRTC clients requesting Opus, and PCM sources such as Wyoming |
| Opus | `PCMA`, `PCMU`, `PCM`, `PCML` | Camera speaker backchannels, playback to G.711 devices, and linear audio consumers |
| Opus | `PCML` then FLAC | MSE/MP4/HLS when a client requests FLAC, for example `?mp4=flac` |

Existing codec matching and source order are preserved. Native input formats precede conversion aliases in each consumer's codec list. An earlier source may be converted even when a later source offers a direct match. Opus is sent with a 48 kHz RTP clock and 20 ms packets. The encoder resamples PCM to 48 kHz and emits CELT packets through the public Go API in `go-opus v0.1.1`. The decoder accepts SILK, CELT and hybrid packets and uses packet loss concealment (PLC) for up to three missing RTP packets. Duplicate and late packets are discarded; larger gaps retain their original timeline. FEC is not requested by this adapter. Stereo conversion preserves separate channel samples.

The conversion does not add AAC, MP3, or video transcoding; use FFmpeg for those codecs. The current `go-opus` runtime no longer depends on `modernc.org/libc`, so the built-in converter also compiles for `linux/mipsle`.

## Integration boundaries

Conversion follows the existing PCM filter pattern: `WithTranscoding` advertises local input aliases, `AddTrack` resolves them to the original protocol format, and `TranscodeHandler` processes RTP packets. These helpers keep no connection state, codec registry or alias cache. Protocol media descriptions and SDP stay separate from local aliases; stream negotiation, playback matching and `core.Node` remain unchanged. Codec instances are owned by packet handlers, and `go-opus` automatically reclaims native resources when those handlers become unreachable. There is no manual TLS, `uintptr` buffer management or codec-specific `Stop` hook in go2rtc.

- WebRTC (including clients that delegate to it), HomeKit, RTSP and WebCodecs can encode PCM/G.711 to Opus.
- MP4/MSE/HLS can decode Opus to linear PCM for the existing FLAC encoder when FLAC is requested. Direct Opus output remains available when the client accepts it. The existing FLAC writer is mono, so stereo input is downmixed for FLAC without changing its sample rate.
- RTSP, Tapo, DVRIP, ISAPI, DoorBird, Wyze, Xiaomi MISS, Tuya and MultiTrans backchannels accept Opus when their output codec is PCM/G.711. An Opus camera backchannel can also accept the standard WebRTC G.711 formats.
- Wyoming, ALSA and exec PCM outputs can decode Opus for playback. Wyoming microphone consumers and WAV backchannels also opt in.

Mono and stereo are supported. A concrete PCM output with omitted channels is mono; an unspecified PCM format, such as the MP4/FLAC input, can preserve source channels. AAC-only backchannels still require FFmpeg.

## Multichannel status

The public Go encoder/decoder API used here handles one or two channels. Multichannel support in go2rtc would also need exported multistream entry points, a channel mapping, and a container or protocol path that carries that mapping. Current WebRTC, HomeKit, and built-in Opus conversion remain mono/stereo; multichannel packets must not be sent through the single-stream path. See [RFC 7845 channel mapping](https://www.rfc-editor.org/rfc/rfc7845.html#section-5.1.1).
