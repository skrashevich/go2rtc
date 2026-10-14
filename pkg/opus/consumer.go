package opus

import (
	"slices"

	"github.com/AlexxIT/go2rtc/pkg/core"
)

// WithTranscoding adds local input formats without modifying protocol medias.
// Like WebRTC's resampling aliases, these are only used to match input tracks.
func WithTranscoding(medias []*core.Media) []*core.Media {
	result := slices.Clone(medias)
	for i, media := range medias {
		if media.Kind != core.KindAudio || media.Direction != core.DirectionSendonly {
			continue
		}
		target := conversionTarget(media.Codecs)
		if target == nil {
			continue
		}
		clone := *media
		clone.Codecs = slices.Clone(media.Codecs)
		names := []string{core.CodecOpus}
		if target.Name == core.CodecOpus {
			names = []string{core.CodecPCMA, core.CodecPCMU, core.CodecPCM, core.CodecPCML}
		}
		for _, name := range names {
			clone.Codecs = append(clone.Codecs, &core.Codec{Name: name})
			// Wildcards only match on the consumer side. Camera backchannels
			// also need the concrete RTP formats used by WebRTC/HomeKit talkers.
			rate := uint32(8000)
			if name == core.CodecOpus {
				rate = 48000
			} else if isPCM(name) {
				continue
			}
			for channels := range uint8(3) {
				clone.Codecs = append(clone.Codecs, &core.Codec{Name: name, ClockRate: rate, Channels: channels})
			}
		}
		result[i] = &clone
	}
	return result
}

func conversionTarget(codecs []*core.Codec) *core.Codec {
	if slices.ContainsFunc(codecs, func(codec *core.Codec) bool { return codec.Name == core.CodecAny || codec.Name == core.CodecAll }) {
		return nil
	}
	var target *core.Codec
	for _, codec := range codecs {
		if codec.Channels > 2 {
			continue
		}
		if codec.Name == core.CodecOpus {
			return codec
		}
		if isLinearOrG711(codec.Name) && (target == nil || codec.Name == core.CodecPCML || isPCM(codec.Name) && !isPCM(target.Name)) {
			target = codec
		}
	}
	return target
}

func isPCM(name string) bool { return name == core.CodecPCM || name == core.CodecPCML }

// AddTrack resolves a conversion alias to the original protocol media/codec.
// Conversion state belongs to the packet handler; no connection-level registry
// or additional lifecycle hooks are needed.
func AddTrack(medias []*core.Media, media *core.Media, codec *core.Codec, track *core.Receiver, add func(*core.Media, *core.Codec, *core.Receiver) error) error {
	for _, original := range medias {
		if original == media {
			return add(media, codec, track)
		}
		if original.ID != media.ID || original.Kind != media.Kind || original.Direction != media.Direction || len(media.Codecs) < len(original.Codecs) || !slices.Equal(original.Codecs, media.Codecs[:len(original.Codecs)]) {
			continue
		}
		if slices.Contains(original.Codecs, codec) {
			return add(original, codec, track)
		}
		if !slices.Contains(media.Codecs[len(original.Codecs):], codec) {
			return add(original, codec, track)
		}
		target := conversionTarget(original.Codecs)
		if target == nil {
			return add(original, codec, track)
		}
		dst := target.Clone()
		if isLinearOrG711(dst.Name) && dst.ClockRate != 0 && dst.Channels == 0 {
			dst.Channels = 1
		}
		dst = outputCodec(dst, track.Codec)
		converted := core.NewReceiver(original, dst)
		handler, err := TranscodeHandler(dst, track.Codec, converted.Input)
		if err != nil {
			return err
		}
		converted.Input = handler
		if err = add(original, target, converted); err != nil {
			return err
		}
		converted.Node.WithParent(&track.Node)
		return nil
	}
	return add(media, codec, track)
}
