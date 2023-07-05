package core

import (
	"fmt"

	"testing"

	"github.com/pion/sdp/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSDP(t *testing.T) {
	medias := []*Media{{
		Kind: KindAudio, Direction: DirectionSendonly,
		Codecs: []*Codec{
			{Name: CodecPCMU, ClockRate: 8000},
		},
	}}

	data, err := MarshalSDP("go2rtc/1.0.0", medias)
	assert.Empty(t, err)

	sd := &sdp.SessionDescription{}
	err = sd.Unmarshal(data)
	assert.Empty(t, err)
}
func TestParseQuery(t *testing.T) {
	query := map[string][]string{
		"video": {"H264,COPY"},
		"audio": {"AAC"},
	}

	expectedMedias := []*Media{
		{
			Kind:      KindVideo,
			Direction: DirectionSendonly,
			Codecs: []*Codec{
				{Name: "H264"},
				{Name: CodecAny},
			},
		},
		{
			Kind:      KindAudio,
			Direction: DirectionSendonly,
			Codecs: []*Codec{
				{Name: CodecAAC},
			},
		},
	}

	medias := ParseQuery(query)

	// Check if the number of medias is correct
	if len(medias) != len(expectedMedias) {
		t.Errorf("Expected %d medias, but got %d", len(expectedMedias), len(medias))
	}

	// Check individual medias
	for i, expectedMedia := range expectedMedias {
		media := medias[i]

		// Check the kind of media
		if media.Kind != expectedMedia.Kind {
			t.Errorf("Expected media kind %s, but got %s", expectedMedia.Kind, media.Kind)
		}

		// Check the direction of media
		if media.Direction != expectedMedia.Direction {
			t.Errorf("Expected media direction %s, but got %s", expectedMedia.Direction, media.Direction)
		}

		// Check the number of codecs in media
		if len(media.Codecs) != len(expectedMedia.Codecs) {
			t.Errorf("Expected %d codecs, but got %d", len(expectedMedia.Codecs), len(media.Codecs))
		}

		// Check individual codecs in media
		for j, expectedCodec := range expectedMedia.Codecs {
			codec := media.Codecs[j]

			// Check the name of codec
			if codec.Name != expectedCodec.Name {
				t.Errorf("Expected codec name %s, but got %s", expectedCodec.Name, codec.Name)
			}
		}
	}
}

func TestClone(t *testing.T) {
	media1 := &Media{
		Kind:      KindVideo,
		Direction: DirectionRecvonly,
		Codecs: []*Codec{
			{Name: CodecPCMU, ClockRate: 8000},
		},
	}
	media2 := media1.Clone()

	p1 := fmt.Sprintf("%p", media1)
	p2 := fmt.Sprintf("%p", media2)
	require.NotEqualValues(t, p1, p2)

	p3 := fmt.Sprintf("%p", media1.Codecs[0])
	p4 := fmt.Sprintf("%p", media2.Codecs[0])
	require.NotEqualValues(t, p3, p4)
}
