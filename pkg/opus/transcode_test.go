package opus

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/pcm"
	"github.com/pion/rtp"
)

func TestTranscodePCMAToOpus(t *testing.T) {
	source := core.NewReceiver(nil, &core.Codec{Name: core.CodecPCMA, ClockRate: 8000, Channels: 1})
	converted, err := TranscodeTrack(source, &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 2})
	if err != nil {
		t.Fatal(err)
	}

	packets := make(chan *rtp.Packet, 2)
	sender := core.NewSender(nil, converted.Codec)
	sender.Handler = func(p *rtp.Packet) { packets <- p }
	sender.HandleRTP(converted)
	defer sender.Close()

	for frame := range 2 {
		payload := make([]byte, 160)
		for i := range payload {
			v := int16(10000 * math.Sin(2*math.Pi*440*float64(frame*160+i)/8000))
			payload[i] = pcm.PCMtoPCMA(v)
		}
		source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: uint16(frame), Timestamp: uint32(frame * 160)}, Payload: payload})
	}

	first := readOpusPacket(t, packets)
	second := readOpusPacket(t, packets)
	header := UnmarshalHeader(first.Payload)
	if header.FrameSize != 20*time.Millisecond || header.Frames != 1 {
		t.Fatalf("Opus packet has %s and %d frames; HomeKit needs one 20ms frame", header.FrameSize, header.Frames)
	}
	if second.Timestamp-first.Timestamp != 960 {
		t.Fatalf("Opus timestamp step = %d, want 960", second.Timestamp-first.Timestamp)
	}
	decoder, err := NewDecoder(48000, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()
	decoded, err := decoder.Decode(first.Payload, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 960*2 {
		t.Fatalf("decoded %d samples, want 1920", len(decoded))
	}
}

func TestCanTranscodeRejectsUnsupportedChannels(t *testing.T) {
	pcma := &core.Codec{Name: core.CodecPCMA, ClockRate: 8000, Channels: 1}
	opus := &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 1}
	if CanTranscode(pcma, &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 3}) {
		t.Fatal("matched unsupported three-channel Opus encoding")
	}
	if CanTranscode(opus, &core.Codec{Name: core.CodecPCML, ClockRate: 48000, Channels: 3}) {
		t.Fatal("matched unsupported three-channel Opus decoding")
	}
}

func TestTranscodePCMAToOpusPreservesTimestampGap(t *testing.T) {
	source := core.NewReceiver(nil, &core.Codec{Name: core.CodecPCMA, ClockRate: 8000, Channels: 1})
	converted, err := TranscodeTrack(source, &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 1})
	if err != nil {
		t.Fatal(err)
	}
	packets := make(chan *rtp.Packet, 2)
	sender := core.NewSender(nil, converted.Codec)
	sender.Handler = func(p *rtp.Packet) { packets <- p }
	sender.HandleRTP(converted)
	defer sender.Close()

	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 1, Timestamp: 0}, Payload: make([]byte, 160)})
	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 3, Timestamp: 320}, Payload: make([]byte, 160)})
	first := readOpusPacket(t, packets)
	second := readOpusPacket(t, packets)
	if second.Timestamp-first.Timestamp != 1920 {
		t.Fatalf("Opus timestamp step across missing source packet = %d, want 1920", second.Timestamp-first.Timestamp)
	}
}

func TestTranscodeWyoming22050PCMToOpus(t *testing.T) {
	source := core.NewReceiver(nil, &core.Codec{Name: core.CodecPCML, ClockRate: 22050, Channels: 1})
	converted, err := TranscodeTrack(source, &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 1})
	if err != nil {
		t.Fatal(err)
	}
	packets := make(chan *rtp.Packet, 50)
	sender := core.NewSender(nil, converted.Codec)
	sender.Handler = func(p *rtp.Packet) { packets <- p }
	sender.HandleRTP(converted)
	defer sender.Close()
	for frame := range 50 {
		source.WriteRTP(&rtp.Packet{Header: rtp.Header{
			Version: 2, SequenceNumber: uint16(frame), Timestamp: uint32(frame * 441),
		}, Payload: make([]byte, 441*2)})
	}
	for range 50 {
		packet := readOpusPacket(t, packets)
		if len(packet.Payload) == 0 {
			t.Fatal("empty Opus packet")
		}
	}
}

func TestTranscodeOpusToPCM(t *testing.T) {
	encoder, err := NewEncoder(16000, 1, ApplicationVoIP)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	input := make([]int16, 320)
	for i := range input {
		input[i] = int16(9000 * math.Sin(2*math.Pi*440*float64(i)/16000))
	}
	packet, err := encoder.Encode(input)
	if err != nil {
		t.Fatal(err)
	}

	source := core.NewReceiver(nil, &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 2})
	converted, err := TranscodeTrack(source, &core.Codec{Name: core.CodecPCML, ClockRate: 16000, Channels: 1})
	if err != nil {
		t.Fatal(err)
	}
	packets := make(chan *rtp.Packet, 2)
	sender := core.NewSender(nil, converted.Codec)
	sender.Handler = func(p *rtp.Packet) { packets <- p }
	sender.HandleRTP(converted)
	defer sender.Close()

	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 7, Timestamp: 2000}, Payload: packet})
	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 8, Timestamp: 2960}, Payload: packet})
	first := readOpusPacket(t, packets)
	second := readOpusPacket(t, packets)
	if len(first.Payload) != 640 {
		t.Fatalf("PCM payload = %d bytes, want 640", len(first.Payload))
	}
	if second.Timestamp-first.Timestamp != 320 {
		t.Fatalf("PCM timestamp step = %d, want 320", second.Timestamp-first.Timestamp)
	}
	var energy int64
	for i := 0; i < len(first.Payload); i += 2 {
		v := int16(binary.LittleEndian.Uint16(first.Payload[i:]))
		energy += int64(v) * int64(v)
	}
	if energy == 0 {
		t.Fatal("decoded PCM is silent")
	}
	sender.Close()
	if source.Senders() != nil {
		t.Fatal("transcoded track remained attached after consumer closed")
	}
}

func TestTranscodeOpusConcealsOneMissingPacket(t *testing.T) {
	encoder, err := NewEncoder(16000, 1, ApplicationVoIP)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	frame := make([]int16, 320)
	for i := range frame {
		frame[i] = int16(9000 * math.Sin(2*math.Pi*440*float64(i)/16000))
	}
	firstData, err := encoder.Encode(frame)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := encoder.Encode(frame); err != nil {
		t.Fatal(err)
	}
	thirdData, err := encoder.Encode(frame)
	if err != nil {
		t.Fatal(err)
	}

	source := core.NewReceiver(nil, &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 2})
	converted, err := TranscodeTrack(source, &core.Codec{Name: core.CodecPCML, ClockRate: 16000, Channels: 1})
	if err != nil {
		t.Fatal(err)
	}
	packets := make(chan *rtp.Packet, 3)
	sender := core.NewSender(nil, converted.Codec)
	sender.Handler = func(p *rtp.Packet) { packets <- p }
	sender.HandleRTP(converted)
	defer sender.Close()

	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 10, Timestamp: 0}, Payload: firstData})
	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 12, Timestamp: 1920}, Payload: thirdData})
	first := readOpusPacket(t, packets)
	missing := readOpusPacket(t, packets)
	third := readOpusPacket(t, packets)
	if missing.Timestamp-first.Timestamp != 320 || third.Timestamp-missing.Timestamp != 320 {
		t.Fatalf("timestamps %d, %d, %d: want 320-sample steps", first.Timestamp, missing.Timestamp, third.Timestamp)
	}
	if len(missing.Payload) != 640 {
		t.Fatalf("concealed frame has %d bytes, want 640", len(missing.Payload))
	}
}

func TestTranscodeOpusConcealsTenMillisecondPacket(t *testing.T) {
	encoder, err := NewEncoder(16000, 1, ApplicationVoIP)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	frame := make([]int16, 160)
	firstData, err := encoder.Encode(frame)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := encoder.Encode(frame); err != nil {
		t.Fatal(err)
	}
	thirdData, err := encoder.Encode(frame)
	if err != nil {
		t.Fatal(err)
	}
	source := core.NewReceiver(nil, &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 1})
	converted, err := TranscodeTrack(source, &core.Codec{Name: core.CodecPCML, ClockRate: 16000, Channels: 1})
	if err != nil {
		t.Fatal(err)
	}
	packets := make(chan *rtp.Packet, 3)
	sender := core.NewSender(nil, converted.Codec)
	sender.Handler = func(p *rtp.Packet) { packets <- p }
	sender.HandleRTP(converted)
	defer sender.Close()
	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 10, Timestamp: 0}, Payload: firstData})
	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 12, Timestamp: 960}, Payload: thirdData})
	first := readOpusPacket(t, packets)
	missing := readOpusPacket(t, packets)
	third := readOpusPacket(t, packets)
	if len(first.Payload) != 320 || len(missing.Payload) != 320 || len(third.Payload) != 320 {
		t.Fatalf("10ms packet lengths: %d, %d, %d; want 320 each", len(first.Payload), len(missing.Payload), len(third.Payload))
	}
	if missing.Timestamp-first.Timestamp != 160 || third.Timestamp-missing.Timestamp != 160 {
		t.Fatalf("10ms timestamps: %d, %d, %d", first.Timestamp, missing.Timestamp, third.Timestamp)
	}
}

func TestTranscodeOpusConcealsTwoTenMillisecondPackets(t *testing.T) {
	encoder, err := NewEncoder(16000, 1, ApplicationVoIP)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	frame := make([]int16, 160)
	firstData, err := encoder.Encode(frame)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := encoder.Encode(frame); err != nil {
			t.Fatal(err)
		}
	}
	fourthData, err := encoder.Encode(frame)
	if err != nil {
		t.Fatal(err)
	}
	source := core.NewReceiver(nil, &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 1})
	converted, err := TranscodeTrack(source, &core.Codec{Name: core.CodecPCML, ClockRate: 16000, Channels: 1})
	if err != nil {
		t.Fatal(err)
	}
	packets := make(chan *rtp.Packet, 4)
	sender := core.NewSender(nil, converted.Codec)
	sender.Handler = func(p *rtp.Packet) { packets <- p }
	sender.HandleRTP(converted)
	defer sender.Close()
	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 10, Timestamp: 0}, Payload: firstData})
	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 13, Timestamp: 1440}, Payload: fourthData})
	for i := range 4 {
		packet := readOpusPacket(t, packets)
		if len(packet.Payload) != 320 {
			t.Fatalf("packet %d has %d bytes, want 320", i, len(packet.Payload))
		}
		if packet.Timestamp != uint32(i*160) {
			t.Fatalf("packet %d timestamp = %d, want %d", i, packet.Timestamp, i*160)
		}
	}
}

func TestTranscodeOpusConcealsSixtyMillisecondPacket(t *testing.T) {
	encoder, err := NewEncoder(16000, 1, ApplicationVoIP)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	frame := make([]int16, 960)
	firstData, err := encoder.Encode(frame)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := encoder.Encode(frame); err != nil {
		t.Fatal(err)
	}
	if _, err := encoder.Encode(frame); err != nil {
		t.Fatal(err)
	}
	fourthData, err := encoder.Encode(frame)
	if err != nil {
		t.Fatal(err)
	}
	source := core.NewReceiver(nil, &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 1})
	converted, err := TranscodeTrack(source, &core.Codec{Name: core.CodecPCML, ClockRate: 16000, Channels: 1})
	if err != nil {
		t.Fatal(err)
	}
	packets := make(chan *rtp.Packet, 4)
	sender := core.NewSender(nil, converted.Codec)
	sender.Handler = func(p *rtp.Packet) { packets <- p }
	sender.HandleRTP(converted)
	defer sender.Close()
	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 10, Timestamp: 0}, Payload: firstData})
	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 13, Timestamp: 8640}, Payload: fourthData})
	for i := range 4 {
		packet := readOpusPacket(t, packets)
		if len(packet.Payload) != 1920 {
			t.Fatalf("packet %d has %d bytes, want 1920", i, len(packet.Payload))
		}
		if packet.Timestamp != uint32(i*960) {
			t.Fatalf("packet %d timestamp = %d, want %d", i, packet.Timestamp, i*960)
		}
	}
}

func TestTranscodeOpusDropsDuplicatePacket(t *testing.T) {
	encoder, err := NewEncoder(16000, 1, ApplicationVoIP)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	data, err := encoder.Encode(make([]int16, 320))
	if err != nil {
		t.Fatal(err)
	}
	source := core.NewReceiver(nil, &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 1})
	converted, err := TranscodeTrack(source, &core.Codec{Name: core.CodecPCML, ClockRate: 16000, Channels: 1})
	if err != nil {
		t.Fatal(err)
	}
	packets := make(chan *rtp.Packet, 3)
	sender := core.NewSender(nil, converted.Codec)
	sender.Handler = func(p *rtp.Packet) { packets <- p }
	sender.HandleRTP(converted)
	defer sender.Close()
	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 10, Timestamp: 0}, Payload: data})
	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 10, Timestamp: 0}, Payload: data})
	source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 11, Timestamp: 960}, Payload: data})
	sender.Close()
	sender.Wait()
	first := readOpusPacket(t, packets)
	second := readOpusPacket(t, packets)
	if second.Timestamp-first.Timestamp != 320 {
		t.Fatalf("timestamp after duplicate = %d, want 320", second.Timestamp-first.Timestamp)
	}
	select {
	case <-packets:
		t.Fatal("duplicate source packet produced decoded audio")
	default:
	}
}

func readOpusPacket(t *testing.T, packets <-chan *rtp.Packet) *rtp.Packet {
	t.Helper()
	select {
	case packet := <-packets:
		return packet
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for converted packet")
		return nil
	}
}
