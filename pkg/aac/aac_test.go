package aac

import (
	"encoding/hex"
	"testing"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/stretchr/testify/assert"
)

func TestConfigToCodec(t *testing.T) {
	s := "profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3;config=F8EC3000"
	s = core.Between(s, "config=", ";")
	src, err := hex.DecodeString(s)
	assert.Nil(t, err)

	codec := ConfigToCodec(src)
	assert.Equal(t, core.CodecAAC, codec.Name)
	assert.Equal(t, uint32(24000), codec.ClockRate)
	assert.Equal(t, uint16(1), codec.Channels)

	dst := EncodeConfig(TypeAACELD, 24000, 1, true)
	assert.Equal(t, src, dst)
}

func TestADTS(t *testing.T) {
	// FFmpeg MPEG-TS AAC (one packet)
	s := "fff15080021ffc210049900219002380fff15080021ffc212049900219002380" //...
	src, err := hex.DecodeString(s)
	assert.Nil(t, err)

	codec := ADTSToCodec(src)
	assert.Equal(t, uint32(44100), codec.ClockRate)
	assert.Equal(t, uint16(2), codec.Channels)

	size := ReadADTSSize(src)
	assert.Equal(t, uint16(16), size)

	dst := CodecToADTS(codec)
	WriteADTSSize(dst, size)

	assert.Equal(t, src[:len(dst)], dst)
}
