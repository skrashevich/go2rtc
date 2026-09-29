package mp4_test

import (
	"bytes"
	"github.com/AlexxIT/go2rtc/internal/streams"
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/mp4"
	"github.com/AlexxIT/go2rtc/pkg/opus"
	"github.com/pion/rtp"
	"sync"
	"testing"
	"time"
)

type opusTestProducer struct{ core.Connection }

func (p *opusTestProducer) Start() error { return nil }

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
	stream := streams.NewStream(nil)
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
