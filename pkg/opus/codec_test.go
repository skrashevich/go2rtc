package opus

import (
	"math"
	"testing"
)

func TestCodecRoundTrip(t *testing.T) {
	encoder, err := NewEncoder(16000, 1, ApplicationVoIP)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()

	decoder, err := NewDecoder(16000, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()

	pcm := make([]int16, 320) // 20 ms at 16 kHz
	for i := range pcm {
		pcm[i] = int16(12000 * math.Sin(2*math.Pi*440*float64(i)/16000))
	}

	packet, err := encoder.Encode(pcm)
	if err != nil {
		t.Fatal(err)
	}
	if len(packet) == 0 {
		t.Fatal("encoder returned an empty Opus packet")
	}

	decoded, err := decoder.Decode(packet, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 320 {
		t.Fatalf("decoded %d samples, want 320", len(decoded))
	}
	var energy int64
	for _, sample := range decoded {
		energy += int64(sample) * int64(sample)
	}
	if energy == 0 {
		t.Fatal("decoded audio is silent")
	}
}

func TestDecoderConcealsMissingFrame(t *testing.T) {
	encoder, err := NewEncoder(48000, 2, ApplicationVoIP)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	packet, err := encoder.Encode(make([]int16, 960*2))
	if err != nil {
		t.Fatal(err)
	}

	decoder, err := NewDecoder(48000, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()
	if _, err := decoder.Decode(packet, false); err != nil {
		t.Fatal(err)
	}

	decoded, err := decoder.Decode(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 960*2 {
		t.Fatalf("decoded %d samples, want 1920", len(decoded))
	}
}

func TestEncoderNetworkControls(t *testing.T) {
	encoder, err := NewEncoder(16000, 1, ApplicationVoIP)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	if err := encoder.SetBitrate(24000); err != nil {
		t.Fatal(err)
	}
	if err := encoder.SetPacketLossPercent(10); err != nil {
		t.Fatal(err)
	}
	if err := encoder.SetInbandFEC(true); err != nil {
		t.Fatal(err)
	}
	if err := encoder.SetDTX(true); err != nil {
		t.Fatal(err)
	}
	if err := encoder.SetPacketLossPercent(101); err == nil {
		t.Fatal("accepted packet loss above 100%")
	}
	if _, err := encoder.Encode(make([]int16, 320)); err != nil {
		t.Fatal(err)
	}
}
