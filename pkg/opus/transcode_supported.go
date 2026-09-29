package opus

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/pcm"
	"github.com/pion/rtp"
)

func CanTranscode(src, dst *core.Codec) bool {
	if src == nil || dst == nil {
		return false
	}
	if src.Name == core.CodecOpus {
		return src.Channels <= 2 && dst.Channels <= 2 && isLinearOrG711(dst.Name)
	}
	return src.ClockRate != 0 && src.Channels <= 2 && dst.Channels <= 2 && isLinearOrG711(src.Name) && dst.Name == core.CodecOpus
}

func isLinearOrG711(name string) bool {
	switch name {
	case core.CodecPCM, core.CodecPCML, core.CodecPCMA, core.CodecPCMU:
		return true
	}
	return false
}

// Track owns a converted receiver and its codec state. Its owner must call Close
// when the consuming connection stops, including when adding the track fails.
type Track struct {
	*core.Receiver
	close func()
}

func (t *Track) Close() { t.close() }

// TranscodeTrack adds a converted receiver below source.
func TranscodeTrack(source *core.Receiver, wanted *core.Codec) (*Track, error) {
	if source == nil || !CanTranscode(source.Codec, wanted) {
		return nil, fmt.Errorf("opus: unsupported conversion")
	}
	if source.Codec.Name == core.CodecOpus {
		return decodeTrack(source, wanted)
	}
	return encodeTrack(source, wanted)
}

func encodeTrack(source *core.Receiver, wanted *core.Codec) (*Track, error) {
	src := source.Codec.Clone()
	if src.Channels == 0 {
		src.Channels = 1
	}
	dst := wanted.Clone()
	dst.ClockRate = 48000 // RFC 7587 RTP clock, independent of encoder input rate
	if dst.Channels == 0 {
		dst.Channels = src.Channels
	}
	if dst.Channels > 2 {
		return nil, fmt.Errorf("opus: unsupported channel count %d", dst.Channels)
	}
	inputRate := nearestOpusRate(src.ClockRate)
	inputCodec := &core.Codec{Name: core.CodecPCML, ClockRate: inputRate, Channels: dst.Channels}
	convert := pcm.Transcode(inputCodec, src)
	encoder, err := NewEncoder(inputRate, dst.Channels, ApplicationVoIP)
	if err != nil {
		return nil, err
	}
	if err = encoder.SetPacketLossPercent(10); err != nil {
		encoder.Close()
		return nil, err
	}
	if err = encoder.SetInbandFEC(true); err != nil {
		encoder.Close()
		return nil, err
	}

	converted := core.NewReceiver(nil, dst)
	forward := converted.Input
	frameSamples := int(inputRate / 50)
	frameBytes := frameSamples * int(dst.Channels) * 2
	var pending []byte
	var sequence uint16
	var timestamp uint32
	var expectedInputTS uint32
	var started bool
	var mu sync.Mutex
	var closed bool
	converted.Input = func(packet *rtp.Packet) {
		mu.Lock()
		defer mu.Unlock()
		if closed {
			return
		}
		if len(packet.Payload)%pcm.BytesPerFrame(src) != 0 {
			return
		}
		if !started || (packet.Timestamp != 0 && packet.Timestamp != expectedInputTS) {
			// Source timestamps describe the audio timeline. Discard any partial
			// frame when a packet was lost or the source jumped in time.
			pending = nil
			timestamp = uint32(uint64(packet.Timestamp) * 48000 / uint64(src.ClockRate))
			started = true
		}
		expectedInputTS = packet.Timestamp + uint32(len(packet.Payload)/pcm.BytesPerFrame(src))
		pending = append(pending, convert(packet.Payload)...)
		for len(pending) >= frameBytes {
			frame := make([]int16, frameSamples*int(dst.Channels))
			for i := range frame {
				frame[i] = int16(binary.LittleEndian.Uint16(pending[i*2:]))
			}
			data, err := encoder.Encode(frame)
			if err != nil {
				return
			}
			out := &rtp.Packet{Header: packet.Header, Payload: data}
			out.SequenceNumber = sequence
			out.Timestamp = timestamp
			out.PayloadType = dst.PayloadType
			forward(out)
			sequence++
			timestamp += 960
			pending = pending[frameBytes:]
		}
	}
	converted.Node.WithParent(&source.Node)
	return &Track{Receiver: converted, close: sync.OnceFunc(func() {
		mu.Lock()
		closed = true
		encoder.Close()
		mu.Unlock()
		converted.Close()
	})}, nil
}

func decodeTrack(source *core.Receiver, wanted *core.Codec) (*Track, error) {
	dst := wanted.Clone()
	if dst.ClockRate == 0 {
		if dst.Name == core.CodecPCMA || dst.Name == core.CodecPCMU {
			dst.ClockRate = 8000
		} else {
			dst.ClockRate = 48000
		}
	}
	if dst.Channels == 0 {
		if dst.Name == core.CodecPCMA || dst.Name == core.CodecPCMU {
			dst.Channels = 1
		} else {
			dst.Channels = source.Codec.Channels
			if dst.Channels == 0 {
				dst.Channels = 1
			}
		}
	}
	if dst.Channels > 2 {
		return nil, fmt.Errorf("opus: unsupported channel count %d", dst.Channels)
	}
	decodeRate := nearestOpusRate(dst.ClockRate)
	decodeCodec := &core.Codec{Name: core.CodecPCML, ClockRate: decodeRate, Channels: dst.Channels}
	convert := pcm.Transcode(dst, decodeCodec)
	decoder, err := NewDecoder(decodeRate, dst.Channels)
	if err != nil {
		return nil, err
	}

	converted := core.NewReceiver(nil, dst)
	forward := converted.Input
	sourceRate := source.Codec.ClockRate
	if sourceRate == 0 {
		sourceRate = 48000
	}
	var sequence uint16
	var timestamp uint32
	var started bool
	var previousInputSeq uint16
	var previousInputTS uint32
	var mu sync.Mutex
	var closed bool
	converted.Input = func(packet *rtp.Packet) {
		mu.Lock()
		defer mu.Unlock()
		if closed {
			return
		}
		if !started {
			timestamp = uint32(uint64(packet.Timestamp) * uint64(dst.ClockRate) / uint64(sourceRate))
			started = true
		} else {
			advance := uint16(packet.SequenceNumber - previousInputSeq)
			if advance == 0 || advance > 0x8000 {
				return // duplicate or late RTP packet
			}
			gap := advance - 1
			if gap > 3 {
				// A large gap or out-of-order packet is a discontinuity, not
				// a request to synthesize unbounded amounts of audio.
				timestamp = uint32(uint64(packet.Timestamp) * uint64(dst.ClockRate) / uint64(sourceRate))
			} else {
				missingSamples := int(uint64(uint32(packet.Timestamp-previousInputTS)) * uint64(decodeRate) / uint64(sourceRate) / uint64(gap+1))
				if missingSamples < 1 || missingSamples > int(decodeRate*120/1000) {
					missingSamples = int(decodeRate / 50)
				}
				for i := range int(gap) {
					var recovered []int16
					var err error
					if i == int(gap)-1 && missingSamples <= int(decodeRate/50) {
						recovered, err = decoder.DecodeFrame(packet.Payload, true, missingSamples)
					} else {
						recovered, err = decoder.DecodeFrame(nil, false, missingSamples)
					}
					if err != nil {
						continue
					}
					emitDecoded(recovered, packet, dst, convert, forward, &sequence, &timestamp)
				}
			}
		}
		previousInputSeq = packet.SequenceNumber
		previousInputTS = packet.Timestamp
		samples, err := decoder.Decode(packet.Payload, false)
		if err != nil {
			return
		}
		emitDecoded(samples, packet, dst, convert, forward, &sequence, &timestamp)
	}
	converted.Node.WithParent(&source.Node)
	return &Track{Receiver: converted, close: sync.OnceFunc(func() {
		mu.Lock()
		closed = true
		decoder.Close()
		mu.Unlock()
		converted.Close()
	})}, nil
}

func emitDecoded(samples []int16, packet *rtp.Packet, dst *core.Codec, convert func([]byte) []byte, forward core.HandlerFunc, sequence *uint16, timestamp *uint32) {
	raw := make([]byte, len(samples)*2)
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(raw[i*2:], uint16(sample))
	}
	payload := convert(raw)
	out := &rtp.Packet{Header: packet.Header, Payload: payload}
	out.SequenceNumber = *sequence
	out.Timestamp = *timestamp
	out.PayloadType = dst.PayloadType
	forward(out)
	*sequence++
	*timestamp += uint32(len(payload) / pcm.BytesPerFrame(dst))
}

func nearestOpusRate(rate uint32) uint32 {
	best := uint32(48000)
	bestDistance := uint32(^uint32(0))
	for _, candidate := range [...]uint32{8000, 12000, 16000, 24000, 48000} {
		var distance uint32
		if rate > candidate {
			distance = rate - candidate
		} else {
			distance = candidate - rate
		}
		if distance < bestDistance {
			best, bestDistance = candidate, distance
		}
	}
	return best
}
