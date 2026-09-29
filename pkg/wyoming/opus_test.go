package wyoming

import (
	"net"
	"testing"
	"time"

	"github.com/AlexxIT/go2rtc/internal/streams"
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/opus"
	"github.com/pion/rtp"
)

type opusSource struct {
	core.Connection
	done chan struct{}
}

func (p *opusSource) Start() error { <-p.done; return nil }
func (p *opusSource) AddTrack(*core.Media, *core.Codec, *core.Receiver) error {
	return core.ErrCantGetTrack
}

func TestOpusPlaybackAndBackchannel(t *testing.T) {
	for _, playback := range []bool{false, true} {
		name := "backchannel"
		if playback {
			name = "playback"
		}
		t.Run(name, func(t *testing.T) {
			local, remote := net.Pipe()
			defer remote.Close()
			_ = remote.SetReadDeadline(time.Now().Add(2 * time.Second))
			camera := newBackchannel(local)
			defer camera.Stop()
			direction := core.DirectionRecvonly
			source := &opusSource{Connection: core.Connection{Medias: []*core.Media{{
				Kind: core.KindAudio, Direction: direction,
				Codecs: []*core.Codec{{Name: core.CodecOpus, ClockRate: 48000, Channels: 2}},
			}}}, done: make(chan struct{})}
			defer close(source.done)
			stream := streams.NewStream(nil)
			stream.AddProducer(camera)
			if playback {
				if err := stream.Play(source); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := stream.AddConsumer(source); err != nil {
					t.Fatal(err)
				}
				defer stream.RemoveConsumer(source)
			}
			encoder, err := opus.NewEncoder(48000, 2, opus.ApplicationVoIP)
			if err != nil {
				t.Fatal(err)
			}
			defer encoder.Close()
			data, err := encoder.Encode(make([]int16, 1920))
			if err != nil {
				t.Fatal(err)
			}
			source.Receivers[0].WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2}, Payload: data})
			event, err := NewAPI(remote).ReadEvent()
			if err != nil {
				t.Fatal(err)
			}
			if event.Type != "audio-chunk" || len(event.Payload) != 882 {
				t.Fatalf("Wyoming audio: %s, %d bytes; want 20ms mono at 22050 Hz", event.Type, len(event.Payload))
			}
			camera.Senders[0].Close()
			camera.Senders[0].Wait()
		})
	}
}
