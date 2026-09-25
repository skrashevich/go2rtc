//go:build !mipsle

package opus

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	goopus "github.com/skrashevich/go-opus"
	"modernc.org/libc"
)

const ApplicationVoIP int32 = 2048

const maxPacketBytes = 1275

const (
	setBitrateRequest    int32 = 4002
	setInbandFECRequest  int32 = 4012
	setPacketLossRequest int32 = 4014
	setDTXRequest        int32 = 4016
)

type Encoder struct {
	mu       sync.Mutex
	tls      *libc.TLS
	state    uintptr
	rate     uint32
	channels uint8
	status   int32
	pcm      []int16
	packet   []byte
}

func NewEncoder(rate uint32, channels uint8, application int32) (*Encoder, error) {
	if !validRate(rate) || channels < 1 || channels > 2 {
		return nil, fmt.Errorf("opus: unsupported encoder format %d Hz/%d channels", rate, channels)
	}
	e := &Encoder{tls: libc.NewTLS(), rate: rate, channels: channels, packet: make([]byte, maxPacketBytes)}
	e.state = goopus.EncoderCreate(e.tls, int32(rate), int32(channels), application, uintptr(unsafe.Pointer(&e.status)))
	if e.state == 0 || e.status != 0 {
		e.tls.Close()
		return nil, fmt.Errorf("opus: encoder creation failed: %d", e.status)
	}
	return e, nil
}

func (e *Encoder) Encode(pcm []int16) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state == 0 {
		return nil, errors.New("opus: encoder is closed")
	}
	if len(pcm)%int(e.channels) != 0 {
		return nil, errors.New("opus: incomplete PCM frame")
	}
	frameSize := len(pcm) / int(e.channels)
	if !validFrameSize(e.rate, frameSize) {
		return nil, fmt.Errorf("opus: invalid frame size %d at %d Hz", frameSize, e.rate)
	}
	// The transpiled codec receives uintptr values. Keep the input on the Go
	// heap so stack growth during the call cannot invalidate its address.
	e.pcm = append(e.pcm[:0], pcm...)
	n := goopus.Encode(e.tls, e.state,
		uintptr(unsafe.Pointer(unsafe.SliceData(e.pcm))), int32(frameSize),
		uintptr(unsafe.Pointer(unsafe.SliceData(e.packet))), int32(len(e.packet)))
	runtime.KeepAlive(e.pcm)
	runtime.KeepAlive(e.packet)
	if n < 0 {
		return nil, fmt.Errorf("opus: encode failed: %d", n)
	}
	return bytes.Clone(e.packet[:n]), nil
}

func (e *Encoder) SetBitrate(bps int32) error {
	return e.setControl(setBitrateRequest, bps)
}

func (e *Encoder) SetPacketLossPercent(percent int32) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("opus: invalid packet loss percentage %d", percent)
	}
	return e.setControl(setPacketLossRequest, percent)
}

func (e *Encoder) SetInbandFEC(enabled bool) error {
	return e.setControl(setInbandFECRequest, boolToInt32(enabled))
}

func (e *Encoder) SetDTX(enabled bool) error {
	return e.setControl(setDTXRequest, boolToInt32(enabled))
}

func (e *Encoder) setControl(request, value int32) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state == 0 {
		return errors.New("opus: encoder is closed")
	}
	va := e.tls.Alloc(16)
	rc := goopus.EncoderCtl(e.tls, e.state, request, libc.VaList(va, value))
	e.tls.Free(16)
	if rc != 0 {
		return fmt.Errorf("opus: encoder control %d failed: %d", request, rc)
	}
	return nil
}

func boolToInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func (e *Encoder) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != 0 {
		goopus.EncoderDestroy(e.tls, e.state)
		e.state = 0
		e.tls.Close()
	}
}

type Decoder struct {
	mu       sync.Mutex
	tls      *libc.TLS
	state    uintptr
	rate     uint32
	channels uint8
	status   int32
	packet   []byte
	pcm      []int16
}

func NewDecoder(rate uint32, channels uint8) (*Decoder, error) {
	if !validRate(rate) || channels < 1 || channels > 2 {
		return nil, fmt.Errorf("opus: unsupported decoder format %d Hz/%d channels", rate, channels)
	}
	d := &Decoder{tls: libc.NewTLS(), rate: rate, channels: channels, pcm: make([]int16, int(rate)*120/1000*int(channels))}
	d.state = goopus.DecoderCreate(d.tls, int32(rate), int32(channels), uintptr(unsafe.Pointer(&d.status)))
	if d.state == 0 || d.status != 0 {
		d.tls.Close()
		return nil, fmt.Errorf("opus: decoder creation failed: %d", d.status)
	}
	return d, nil
}

func (d *Decoder) Decode(packet []byte, fec bool) ([]int16, error) {
	return d.DecodeFrame(packet, fec, 0)
}

// DecodeFrame decodes a packet or conceals a missing frame. For concealment,
// samples is the requested number of samples per channel; zero uses 20 ms.
func (d *Decoder) DecodeFrame(packet []byte, fec bool, samples int) ([]int16, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state == 0 {
		return nil, errors.New("opus: decoder is closed")
	}
	var data uintptr
	frameSize := int32(d.rate * 120 / 1000)
	if len(packet) == 0 || fec {
		if samples == 0 {
			samples = int(d.rate / 50)
		}
		if samples < 1 || samples > int(d.rate*120/1000) {
			return nil, fmt.Errorf("opus: invalid concealment frame size %d", samples)
		}
		frameSize = int32(samples)
	}
	if len(packet) != 0 {
		d.packet = append(d.packet[:0], packet...)
		data = uintptr(unsafe.Pointer(unsafe.SliceData(d.packet)))
	}
	var fecFlag int32
	if fec {
		fecFlag = 1
	}
	n := goopus.Decode(d.tls, d.state, data, int32(len(packet)),
		uintptr(unsafe.Pointer(unsafe.SliceData(d.pcm))), frameSize, fecFlag)
	runtime.KeepAlive(d.packet)
	runtime.KeepAlive(d.pcm)
	if n < 0 {
		return nil, fmt.Errorf("opus: decode failed: %d", n)
	}
	return append([]int16(nil), d.pcm[:int(n)*int(d.channels)]...), nil
}

func (d *Decoder) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state != 0 {
		goopus.DecoderDestroy(d.tls, d.state)
		d.state = 0
		d.tls.Close()
	}
}

func validRate(rate uint32) bool {
	switch rate {
	case 8000, 12000, 16000, 24000, 48000:
		return true
	}
	return false
}

func validFrameSize(rate uint32, samples int) bool {
	for _, duration400 := range [...]int{1, 2, 4, 8, 16, 24} {
		if samples*400 == int(rate)*duration400 {
			return true
		}
	}
	return false
}
