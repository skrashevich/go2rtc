//go:build mipsle

package opus

import (
	"errors"

	"github.com/AlexxIT/go2rtc/pkg/core"
)

const ApplicationVoIP int32 = 2048

var errUnsupportedPlatform = errors.New("opus: codec unavailable on mipsle (modernc.org/libc has no mipsle support)")

type Encoder struct{}

func NewEncoder(uint32, uint8, int32) (*Encoder, error) { return nil, errUnsupportedPlatform }
func (*Encoder) Encode([]int16) ([]byte, error)         { return nil, errUnsupportedPlatform }
func (*Encoder) SetBitrate(int32) error                 { return errUnsupportedPlatform }
func (*Encoder) SetPacketLossPercent(int32) error       { return errUnsupportedPlatform }
func (*Encoder) SetInbandFEC(bool) error                { return errUnsupportedPlatform }
func (*Encoder) SetDTX(bool) error                      { return errUnsupportedPlatform }
func (*Encoder) Close()                                 {}

type Decoder struct{}

func NewDecoder(uint32, uint8) (*Decoder, error) { return nil, errUnsupportedPlatform }
func (*Decoder) Decode([]byte, bool) ([]int16, error) {
	return nil, errUnsupportedPlatform
}
func (*Decoder) DecodeFrame([]byte, bool, int) ([]int16, error) {
	return nil, errUnsupportedPlatform
}
func (*Decoder) Close() {}

func CanTranscode(*core.Codec, *core.Codec) bool { return false }
func TranscodeTrack(*core.Receiver, *core.Codec) (*core.Receiver, error) {
	return nil, errUnsupportedPlatform
}
