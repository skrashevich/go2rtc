package homekit

import (
	"net"
	"testing"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/opus"
	"github.com/AlexxIT/go2rtc/pkg/srtp"
	"github.com/pion/rtp"
)

func TestConsumerEncodesOpus(t *testing.T) {
	local, remote := net.Pipe()
	defer remote.Close()
	c := NewConsumer(local, nil)
	t.Cleanup(func() { _ = c.Stop() })
	c.audioSession = &srtp.Session{Local: &srtp.Endpoint{}}
	c.audioRTPTime = 20
	input := &core.Media{Kind: core.KindAudio, Direction: core.DirectionRecvonly,
		Codecs: []*core.Codec{{Name: core.CodecPCML, ClockRate: 16000, Channels: 1}}}
	media := c.GetMedias()[1]
	_, codec := input.MatchMedia(media)
	if codec == nil {
		t.Fatal("PCM input not advertised")
	}
	source := core.NewReceiver(input, input.Codecs[0])
	if err := c.AddTrack(media, codec, source); err != nil {
		t.Fatal(err)
	}
	if len(c.Medias[1].Codecs) != 1 || c.Senders[0].Codec.Name != core.CodecOpus {
		t.Fatal("wrong HomeKit media codec")
	}
	output := make(chan *rtp.Packet, 1)
	sender := c.Senders[0]
	original := sender.Handler
	sender.Handler = func(p *rtp.Packet) { original(p); output <- p }
	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2}, Payload: make([]byte, 640)})
	select {
	case packet := <-output:
		header := opus.UnmarshalHeader(packet.Payload)
		if header.FrameSize != 20*time.Millisecond || header.Frames != 1 {
			t.Fatalf("unsupported HomeKit packet: %+v", header)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no HomeKit Opus output")
	}
	sender.Close()
	sender.Wait()
}
