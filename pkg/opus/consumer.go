package opus

import (
	"errors"
	"slices"
	"sync"

	"github.com/AlexxIT/go2rtc/pkg/core"
)

// Consumer adds Opus conversion to a consumer that explicitly opts in. The
// consumer must call Close when it stops. Original media descriptions remain
// unchanged, so conversion capabilities never leak into protocol negotiation.
type Consumer struct {
	mu      sync.Mutex
	medias  map[*core.Media]*core.Media
	views   map[*core.Media]consumerView
	targets map[*core.Codec]*core.Codec
	aliases map[*core.Codec][]*core.Codec
	tracks  []*Track
	closed  bool
}

type consumerView struct {
	media  *core.Media
	codecs []*core.Codec
}

// GetMedias advertises additional input formats for sendonly audio media.
// Existing codecs precede conversion aliases; source ordering is unchanged.
func (c *Consumer) GetMedias(medias []*core.Media) []*core.Media {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.medias == nil {
		c.medias = make(map[*core.Media]*core.Media)
		c.views = make(map[*core.Media]consumerView)
		c.targets = make(map[*core.Codec]*core.Codec)
		c.aliases = make(map[*core.Codec][]*core.Codec)
	}
	result := make([]*core.Media, len(medias))
	for i, media := range medias {
		result[i] = media
		if media.Kind != core.KindAudio || media.Direction != core.DirectionSendonly {
			continue
		}
		if view, ok := c.views[media]; ok && view.media.ID == media.ID && slices.Equal(view.codecs, media.Codecs) {
			result[i] = view.media
			continue
		}
		var encoder, decoder *core.Codec
		for _, codec := range media.Codecs {
			if codec.Channels > 2 {
				continue
			}
			switch codec.Name {
			case core.CodecOpus:
				if encoder == nil {
					encoder = codec
				}
			case core.CodecPCML, core.CodecPCM:
				if decoder == nil || !isPCM(decoder.Name) || codec.Name == core.CodecPCML {
					decoder = codec
				}
			case core.CodecPCMA, core.CodecPCMU:
				if decoder == nil {
					decoder = codec
				}
			}
		}
		clone := *media
		clone.Codecs = append([]*core.Codec(nil), media.Codecs...)
		if encoder != nil {
			clone.Codecs = append(clone.Codecs, c.inputs(encoder, core.CodecPCMA, core.CodecPCMU, core.CodecPCM, core.CodecPCML)...)
		} else if decoder != nil {
			clone.Codecs = append(clone.Codecs, c.inputs(decoder, core.CodecOpus)...)
		}
		if len(clone.Codecs) != len(media.Codecs) {
			// Published views are immutable; another consumer may still be
			// matching an earlier view when protocol capabilities change.
			c.medias[&clone] = media
			c.views[media] = consumerView{media: &clone, codecs: slices.Clone(media.Codecs)}
			result[i] = &clone
		}
	}
	return result
}

func isPCM(name string) bool { return name == core.CodecPCM || name == core.CodecPCML }

func (c *Consumer) inputs(target *core.Codec, names ...string) []*core.Codec {
	if aliases := c.aliases[target]; aliases != nil {
		return aliases
	}
	var aliases []*core.Codec
	for _, name := range names {
		alias := &core.Codec{Name: name}
		c.targets[alias] = target
		aliases = append(aliases, alias)
		// MatchMedia compares a camera's sendonly codecs against the remote
		// talker's codecs. Wildcards only work on the right side, so include
		// the concrete RTP formats used by WebRTC and HomeKit backchannels.
		var rate uint32
		switch name {
		case core.CodecOpus:
			rate = 48000
		case core.CodecPCMA, core.CodecPCMU:
			rate = 8000
		}
		if rate != 0 {
			for channels := range uint8(3) {
				alias := &core.Codec{Name: name, ClockRate: rate, Channels: channels}
				c.targets[alias] = target
				aliases = append(aliases, alias)
			}
		}
	}
	c.aliases[target] = aliases
	return aliases
}

// AddTrack resolves a local conversion alias before calling the protocol's
// normal AddTrack implementation with its original media and output codec.
func (c *Consumer) AddTrack(media *core.Media, codec *core.Codec, track *core.Receiver, add func(*core.Media, *core.Codec, *core.Receiver) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("opus: consumer closed")
	}
	if original := c.medias[media]; original != nil {
		media = original
	}
	target := c.targets[codec]
	if target == nil {
		return add(media, codec, track)
	}
	wanted := target
	if isLinearOrG711(target.Name) && target.ClockRate != 0 && target.Channels == 0 {
		// Concrete PCM formats use omitted channels for mono. A fully
		// unspecified PCM format (e.g. MP4/FLAC) can retain source channels.
		wanted = target.Clone()
		wanted.Channels = 1
	}
	converted, err := TranscodeTrack(track, wanted)
	if err != nil {
		return err
	}
	if err = add(media, target, converted.Receiver); err != nil {
		converted.Close()
		return err
	}
	c.tracks = append(c.tracks, converted)
	return nil
}

func (c *Consumer) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	for _, track := range c.tracks {
		track.Close()
	}
	c.tracks = nil
}
