package opus

import (
	"errors"
	"testing"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/pion/rtp"
)

func TestConsumerLocalCapabilities(t *testing.T) {
	for _, name := range []string{core.CodecOpus, core.CodecPCMA, core.CodecPCMU, core.CodecPCM, core.CodecPCML} {
		t.Run(name, func(t *testing.T) {
			var c Consumer
			t.Cleanup(c.Close)
			codec := &core.Codec{Name: name, ClockRate: 48000, Channels: 2}
			media := &core.Media{Kind: core.KindAudio, Direction: core.DirectionSendonly, Codecs: []*core.Codec{codec}}
			recv := &core.Media{Kind: core.KindAudio, Direction: core.DirectionRecvonly, Codecs: []*core.Codec{codec}}
			medias := c.GetMedias([]*core.Media{media, recv})
			if len(media.Codecs) != 1 || medias[1] != recv {
				t.Fatal("changed protocol media descriptions")
			}
			if medias[0].Codecs[0] != codec {
				t.Fatal("changed direct codec preference")
			}
			again := c.GetMedias([]*core.Media{media})
			if again[0] != medias[0] || again[0].Codecs[1] != medias[0].Codecs[1] {
				t.Fatal("unstable aliases")
			}
			input := core.CodecOpus
			rate := uint32(48000)
			if name == core.CodecOpus {
				input = core.CodecPCMA
				rate = 8000
			}
			for channels := range uint8(3) {
				source := &core.Media{Kind: core.KindAudio, Direction: core.DirectionRecvonly,
					Codecs: []*core.Codec{{Name: input, ClockRate: rate, Channels: channels}}}
				for _, backchannel := range []bool{false, true} {
					var local *core.Codec
					if backchannel {
						local, _ = medias[0].MatchMedia(source)
					} else {
						_, local = source.MatchMedia(medias[0])
					}
					if local == nil || c.targets[local] != codec {
						t.Fatalf("no conversion for channels=%d backchannel=%v", channels, backchannel)
					}
				}
			}
		})
	}
}

func TestConsumerDirectAndLinearPreference(t *testing.T) {
	var c Consumer
	t.Cleanup(c.Close)
	pcm := &core.Codec{Name: core.CodecPCML}
	media := &core.Media{Kind: core.KindAudio, Direction: core.DirectionSendonly,
		Codecs: []*core.Codec{{Name: core.CodecPCMA}, {Name: core.CodecPCM}, pcm}}
	local := c.GetMedias([]*core.Media{media})[0]
	input := &core.Media{Kind: core.KindAudio, Direction: core.DirectionRecvonly,
		Codecs: []*core.Codec{{Name: core.CodecOpus, ClockRate: 48000, Channels: 2}}}
	_, alias := input.MatchMedia(local)
	if c.targets[alias] != pcm {
		t.Fatal("Opus decoding should prefer linear PCM for FLAC")
	}
	input.Codecs[0] = &core.Codec{Name: core.CodecPCMA, ClockRate: 8000}
	_, native := input.MatchMedia(local)
	source := core.NewReceiver(input, input.Codecs[0])
	if err := c.AddTrack(local, native, source, func(m *core.Media, codec *core.Codec, track *core.Receiver) error {
		if m != media || codec != media.Codecs[0] || track != source {
			t.Fatal("direct track was changed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(c.tracks) != 0 {
		t.Fatal("allocated codec for direct track")
	}
}

func TestConsumerConversionLifetime(t *testing.T) {
	var c Consumer
	media := &core.Media{Kind: core.KindAudio, Direction: core.DirectionSendonly,
		Codecs: []*core.Codec{{Name: core.CodecOpus, ClockRate: 48000, Channels: 2}}}
	local := c.GetMedias([]*core.Media{media})[0]
	input := &core.Media{Kind: core.KindAudio, Direction: core.DirectionRecvonly,
		Codecs: []*core.Codec{{Name: core.CodecPCMA, ClockRate: 8000}}}
	source := core.NewReceiver(input, input.Codecs[0])
	_, alias := input.MatchMedia(local)
	rejected := errors.New("rejected")
	if err := c.AddTrack(local, alias, source, func(*core.Media, *core.Codec, *core.Receiver) error { return rejected }); !errors.Is(err, rejected) {
		t.Fatalf("AddTrack error = %v", err)
	}
	if len(c.tracks) != 0 || len(source.Senders()) != 0 {
		t.Fatal("failed AddTrack retained a converter")
	}
	var converted *core.Receiver
	var packet *rtp.Packet
	if err := c.AddTrack(local, alias, source, func(m *core.Media, codec *core.Codec, track *core.Receiver) error {
		if m != media || codec != media.Codecs[0] || track.Codec.Name != core.CodecOpus {
			t.Fatal("wrong protocol output")
		}
		converted = track
		track.AppendChild(&core.Node{Input: func(p *rtp.Packet) { packet = p }})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	inputPacket := &rtp.Packet{Header: rtp.Header{Version: 2}, Payload: make([]byte, 160)}
	source.WriteRTP(inputPacket)
	if packet == nil {
		t.Fatal("no encoded audio")
	}
	d, err := NewDecoder(48000, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	samples, err := d.Decode(packet.Payload, false)
	if err != nil || len(samples) != 1920 {
		t.Fatalf("decode: %d samples, %v", len(samples), err)
	}
	c.Close()
	c.Close()
	packet = nil
	converted.WriteRTP(inputPacket)
	if packet != nil || len(c.tracks) != 0 {
		t.Fatal("converter remains active after Close")
	}
	if err = c.AddTrack(local, alias, source, nil); err == nil {
		t.Fatal("accepted track after Close")
	}
}
