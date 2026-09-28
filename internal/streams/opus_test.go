package streams

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/mp4"
	"github.com/AlexxIT/go2rtc/pkg/opus"
	"github.com/pion/rtp"
)

type opusTestProducer struct {
	core.Connection
	packets chan *rtp.Packet
}

func (p *opusTestProducer) Start() error { return nil }

type opusTestConsumer struct {
	core.Connection
	packets chan *rtp.Packet
	track   *core.Receiver
}

type opusTestWriter struct {
	bytes.Buffer
	mu     sync.Mutex
	writes int
	init   chan struct{}
	media  chan struct{}
}

func (w *opusTestWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writes++
	if w.writes == 1 {
		close(w.init)
	}
	if w.writes == 2 {
		close(w.media)
	}
	return w.Buffer.Write(p)
}

func (c *opusTestConsumer) AddTrack(media *core.Media, codec *core.Codec, track *core.Receiver) error {
	c.track = track
	sender := core.NewSender(media, codec)
	sender.Handler = func(packet *rtp.Packet) { c.packets <- packet }
	sender.HandleRTP(track)
	c.Senders = append(c.Senders, sender)
	return nil
}

func (c *opusTestConsumer) Start() error { return nil }

func (p *opusTestProducer) AddTrack(media *core.Media, codec *core.Codec, track *core.Receiver) error {
	sender := core.NewSender(media, codec)
	sender.Handler = func(packet *rtp.Packet) { p.packets <- packet }
	sender.HandleRTP(track)
	p.Senders = append(p.Senders, sender)
	return nil
}

func TestAddConsumerTranscodesPCMAToOpus(t *testing.T) {
	srcCodec := &core.Codec{Name: core.CodecPCMA, ClockRate: 8000, Channels: 1}
	prod := &opusTestProducer{Connection: core.Connection{Medias: []*core.Media{{
		Kind: core.KindAudio, Direction: core.DirectionRecvonly, Codecs: []*core.Codec{srcCodec},
	}}}}
	dstCodec := &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 2}
	cons := &opusTestConsumer{Connection: core.Connection{Medias: []*core.Media{{
		Kind: core.KindAudio, Direction: core.DirectionSendonly, Codecs: []*core.Codec{dstCodec},
	}}}, packets: make(chan *rtp.Packet, 1)}
	stream := NewStream(nil)
	stream.AddProducer(prod)
	if err := stream.AddConsumer(cons); err != nil {
		t.Fatal(err)
	}
	defer stream.RemoveConsumer(cons)

	prod.Receivers[0].WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2}, Payload: make([]byte, 160)})
	select {
	case packet := <-cons.packets:
		decoder, err := opus.NewDecoder(48000, 2)
		if err != nil {
			t.Fatal(err)
		}
		defer decoder.Close()
		decoded, err := decoder.Decode(packet.Payload, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(decoded) != 960*2 {
			t.Fatalf("decoded %d samples, want 1920", len(decoded))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Opus")
	}
}

func TestAddConsumerPrefersDirectOpusFromLaterProducer(t *testing.T) {
	pcma := &opusTestProducer{Connection: core.Connection{Medias: []*core.Media{{
		Kind: core.KindAudio, Direction: core.DirectionRecvonly,
		Codecs: []*core.Codec{{Name: core.CodecPCMA, ClockRate: 8000}},
	}}}}
	opusCodec := &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 2}
	direct := &opusTestProducer{Connection: core.Connection{Medias: []*core.Media{{
		Kind: core.KindAudio, Direction: core.DirectionRecvonly, Codecs: []*core.Codec{opusCodec},
	}}}}
	cons := &opusTestConsumer{Connection: core.Connection{Medias: []*core.Media{{
		Kind: core.KindAudio, Direction: core.DirectionSendonly,
		Codecs: []*core.Codec{{Name: core.CodecOpus, ClockRate: 48000, Channels: 2}},
	}}}, packets: make(chan *rtp.Packet, 1)}
	stream := NewStream(nil)
	stream.AddProducer(pcma)
	stream.AddProducer(direct)
	if err := stream.AddConsumer(cons); err != nil {
		t.Fatal(err)
	}
	defer stream.RemoveConsumer(cons)
	if cons.track.Codec != opusCodec {
		t.Fatal("selected transcoding although an Opus track was available")
	}
}

func TestAddConsumerTranscodesOpusBackchannelToPCMA(t *testing.T) {
	encoder, err := opus.NewEncoder(16000, 1, opus.ApplicationVoIP)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	data, err := encoder.Encode(make([]int16, 320))
	if err != nil {
		t.Fatal(err)
	}

	backchannel := &opusTestProducer{Connection: core.Connection{Medias: []*core.Media{{
		Kind: core.KindAudio, Direction: core.DirectionSendonly,
		Codecs: []*core.Codec{{Name: core.CodecPCMA, ClockRate: 8000, Channels: 1}},
	}}}, packets: make(chan *rtp.Packet, 1)}
	cons := &opusTestConsumer{Connection: core.Connection{Medias: []*core.Media{{
		Kind: core.KindAudio, Direction: core.DirectionRecvonly,
		Codecs: []*core.Codec{{Name: core.CodecOpus, ClockRate: 48000, Channels: 2}},
	}}}, packets: make(chan *rtp.Packet, 1)}
	stream := NewStream(nil)
	stream.AddProducer(backchannel)
	if err := stream.AddConsumer(cons); err != nil {
		t.Fatal(err)
	}
	defer stream.RemoveConsumer(cons)
	cons.Receivers[0].WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2}, Payload: data})
	select {
	case packet := <-backchannel.packets:
		if len(packet.Payload) != 160 {
			t.Fatalf("PCMA payload = %d bytes, want 160", len(packet.Payload))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PCMA backchannel audio")
	}
}

func TestOpusToFLACSelectsLosslessPCM(t *testing.T) {
	producer := &core.Media{Kind: core.KindAudio, Direction: core.DirectionRecvonly,
		Codecs: []*core.Codec{{Name: core.CodecOpus, ClockRate: 48000, Channels: 2}}}
	consumer := mp4.ParseQuery(map[string][]string{"mp4": {"flac"}})[1]
	_, target := matchTranscodedMedia(producer, consumer)
	if target == nil || target.Name != core.CodecPCML {
		t.Fatalf("selected %v, want PCML for lossless FLAC conversion", target)
	}
}

func TestAddConsumerOpusToMP4FLAC(t *testing.T) {
	encoder, err := opus.NewEncoder(16000, 1, opus.ApplicationVoIP)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	data, err := encoder.Encode(make([]int16, 320))
	if err != nil {
		t.Fatal(err)
	}
	prod := &opusTestProducer{Connection: core.Connection{Medias: []*core.Media{{
		Kind: core.KindAudio, Direction: core.DirectionRecvonly,
		Codecs: []*core.Codec{{Name: core.CodecOpus, ClockRate: 48000, Channels: 1}},
	}}}}
	cons := mp4.NewConsumer(mp4.ParseQuery(map[string][]string{"mp4": {"flac"}}))
	stream := NewStream(nil)
	stream.AddProducer(prod)
	if err := stream.AddConsumer(cons); err != nil {
		t.Fatal(err)
	}
	defer stream.RemoveConsumer(cons)
	if len(cons.Senders) != 1 || cons.Senders[0].Codec.Name != core.CodecFLAC {
		t.Fatalf("MP4 senders = %v; want FLAC", cons.Senders)
	}
	output := &opusTestWriter{init: make(chan struct{}), media: make(chan struct{})}
	done := make(chan error, 1)
	go func() { _, err := cons.WriteTo(output); done <- err }()
	select {
	case <-output.init:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out writing MP4 init")
	}
	prod.Receivers[0].WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2}, Payload: data})
	select {
	case <-output.media:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for FLAC media payload")
	}
	cons.Stop()
	cons.Senders[0].Wait()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out writing MP4")
	}
	if cons.Send == 0 {
		t.Fatal("Opus to FLAC conversion produced no MP4 media payload")
	}
}

func TestPlayMatchesOpusSourceToPCMACamera(t *testing.T) {
	encoder, err := opus.NewEncoder(16000, 1, opus.ApplicationVoIP)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	data, err := encoder.Encode(make([]int16, 320))
	if err != nil {
		t.Fatal(err)
	}
	source := &opusTestProducer{Connection: core.Connection{Medias: []*core.Media{{
		Kind: core.KindAudio, Direction: core.DirectionRecvonly,
		Codecs: []*core.Codec{{Name: core.CodecOpus, ClockRate: 48000, Channels: 2}},
	}}}}
	camera := &opusTestConsumer{Connection: core.Connection{Medias: []*core.Media{{
		Kind: core.KindAudio, Direction: core.DirectionSendonly,
		Codecs: []*core.Codec{{Name: core.CodecPCMA, ClockRate: 8000, Channels: 1}},
	}}}, packets: make(chan *rtp.Packet, 1)}
	if !matchMedia(source, camera) {
		t.Fatal("Opus source could not play to PCMA camera")
	}
	defer camera.Stop()
	source.Receivers[0].WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2}, Payload: data})
	select {
	case packet := <-camera.packets:
		if len(packet.Payload) != 160 {
			t.Fatalf("PCMA payload = %d bytes, want 160", len(packet.Payload))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for camera audio")
	}
}
