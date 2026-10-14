package opus

import (
	"errors"
	"testing"

	"github.com/AlexxIT/go2rtc/pkg/core"
)

func TestLocalCapabilities(t *testing.T) {
	for _, name := range []string{core.CodecOpus, core.CodecPCMA, core.CodecPCMU, core.CodecPCM, core.CodecPCML} {
		t.Run(name, func(t *testing.T) {
			codec := &core.Codec{Name: name, ClockRate: 48000, Channels: 2}
			media := &core.Media{Kind: core.KindAudio, Direction: core.DirectionSendonly, Codecs: []*core.Codec{codec}}
			recv := &core.Media{Kind: core.KindAudio, Direction: core.DirectionRecvonly, Codecs: []*core.Codec{codec}}
			local := WithTranscoding([]*core.Media{media, recv})
			if len(media.Codecs) != 1 || local[1] != recv || local[0].Codecs[0] != codec {
				t.Fatal("changed protocol media or direct codec")
			}
			input, rate := core.CodecOpus, uint32(48000)
			if name == core.CodecOpus {
				input, rate = core.CodecPCMA, 8000
			}
			for channels := range uint8(3) {
				source := &core.Media{Kind: core.KindAudio, Direction: core.DirectionRecvonly, Codecs: []*core.Codec{{Name: input, ClockRate: rate, Channels: channels}}}
				for _, reverse := range []bool{false, true} {
					var alias *core.Codec
					if reverse {
						alias, _ = local[0].MatchMedia(source)
					} else {
						_, alias = source.MatchMedia(local[0])
					}
					if alias == nil {
						t.Fatalf("no alias for channels=%d reverse=%v", channels, reverse)
					}
				}
			}
		})
	}
}

func TestConversionRejectsMultichannel(t *testing.T) {
	media := &core.Media{Kind: core.KindAudio, Direction: core.DirectionSendonly, Codecs: []*core.Codec{{Name: core.CodecOpus, ClockRate: 48000, Channels: 2}}}
	local := WithTranscoding([]*core.Media{media})[0]
	input := &core.Media{Kind: core.KindAudio, Direction: core.DirectionRecvonly, Codecs: []*core.Codec{{Name: core.CodecPCML, ClockRate: 48000, Channels: 3}}}
	_, alias := input.MatchMedia(local)
	if alias == nil {
		t.Fatal("missing wildcard alias")
	}
	err := AddTrack([]*core.Media{media}, local, alias, core.NewReceiver(input, input.Codecs[0]), func(*core.Media, *core.Codec, *core.Receiver) error {
		t.Fatal("forwarded unconverted multichannel PCM to Opus")
		return nil
	})
	if err == nil {
		t.Fatal("accepted unsupported multichannel input")
	}
}

func TestDirectAndWildcardPassthrough(t *testing.T) {
	for _, name := range []string{core.CodecPCMA, core.CodecOpus, core.CodecAny, core.CodecAll} {
		codec := &core.Codec{Name: name}
		media := &core.Media{Kind: core.KindAudio, Direction: core.DirectionSendonly, Codecs: []*core.Codec{codec}}
		source := core.NewReceiver(nil, &core.Codec{Name: core.CodecPCMA, ClockRate: 8000})
		local := WithTranscoding([]*core.Media{media})[0]
		if err := AddTrack([]*core.Media{media}, local, codec, source, func(m *core.Media, c *core.Codec, r *core.Receiver) error {
			if m != media || c != codec || r != source {
				t.Fatal("changed a native track")
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestConversionAttachAndDetach(t *testing.T) {
	for _, dst := range []*core.Codec{{Name: core.CodecOpus, ClockRate: 48000, Channels: 2}, {Name: core.CodecPCML, ClockRate: 22050}} {
		media := &core.Media{Kind: core.KindAudio, Direction: core.DirectionSendonly, Codecs: []*core.Codec{dst}}
		local := WithTranscoding([]*core.Media{media})[0]
		src := &core.Codec{Name: core.CodecPCMA, ClockRate: 8000}
		data := make([]byte, 160)
		if dst.Name != core.CodecOpus {
			src = &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 2}
			data = encodeTestPacket(t, 2)
		}
		input := &core.Media{Kind: core.KindAudio, Direction: core.DirectionRecvonly, Codecs: []*core.Codec{src}}
		source := core.NewReceiver(input, src)
		_, alias := input.MatchMedia(local)
		rejected := errors.New("rejected")
		if err := AddTrack([]*core.Media{media}, local, alias, source, func(*core.Media, *core.Codec, *core.Receiver) error { return rejected }); !errors.Is(err, rejected) {
			t.Fatal(err)
		}
		if len(source.Senders()) != 0 {
			t.Fatal("failed AddTrack attached a filter")
		}
		var converted *core.Receiver
		var packet *core.Packet
		if err := AddTrack([]*core.Media{media}, local, alias, source, func(m *core.Media, c *core.Codec, r *core.Receiver) error {
			if m != media || c != dst || r.Codec.Name != dst.Name {
				t.Fatal("wrong protocol output")
			}
			converted = r
			r.AppendChild(&core.Node{Input: func(p *core.Packet) { packet = p }})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		source.WriteRTP(&core.Packet{Header: packetHeader(0), Payload: data})
		if packet == nil {
			t.Fatal("no converted output")
		}
		if dst.Name == core.CodecPCML && (len(packet.Payload) != 882 || converted.Codec.Channels != 1) {
			t.Fatal("concrete PCM output must be mono")
		}
		converted.Close()
		packet = nil
		source.WriteRTP(&core.Packet{Header: packetHeader(960), Payload: data})
		if packet != nil || len(source.Senders()) != 0 {
			t.Fatal("filter remains attached after track Close")
		}
	}
}
