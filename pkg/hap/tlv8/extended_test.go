package tlv8

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBooleanWireFormat(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		v := struct {
			Enabled bool `tlv8:"1"`
		}{enabled}
		wire, err := Marshal(v)
		require.NoError(t, err)
		want := byte(0)
		if enabled {
			want = 1
		}
		require.Equal(t, []byte{1, 1, want}, wire)
		var decoded struct {
			Enabled bool `tlv8:"1"`
		}
		require.NoError(t, Unmarshal(wire, &decoded))
		require.Equal(t, v, decoded)
	}
}

func TestBooleanRejectsMalformedValue(t *testing.T) {
	for _, wire := range [][]byte{{1, 1, 2}, {1, 2, 0, 1}} {
		var v struct {
			Enabled bool `tlv8:"1"`
		}
		require.Error(t, Unmarshal(wire, &v))
	}
}

func TestNestedTLVFragments(t *testing.T) {
	type inner struct {
		SDP string `tlv8:"1"`
	}
	type outer struct {
		Offer  inner `tlv8:"2"`
		Status byte  `tlv8:"3"`
	}
	// Inner encoding has 252 payload bytes + 2 header bytes = 254;
	// exercise both sides of the outer fragment boundary, and multiple fragments.
	for _, size := range []int{252, 253, 254, 255, 256, 508, 509, 510, 1024} {
		v := outer{Offer: inner{SDP: strings.Repeat("s", size)}, Status: 7}
		wire, err := Marshal(v)
		require.NoError(t, err)
		var got outer
		require.NoError(t, Unmarshal(wire, &got), "size %d", size)
		require.Equal(t, v, got)
	}
	// Independently calculated wire for 256 bytes of nested content.
	v := outer{Offer: inner{SDP: strings.Repeat("x", 254)}, Status: 7}
	wire, err := Marshal(v)
	require.NoError(t, err)
	want := append([]byte{2, 255, 1, 254}, bytes.Repeat([]byte{'x'}, 253)...)
	want = append(want, 2, 1, 'x', 3, 1, 7)
	require.Equal(t, want, wire)
}

func TestArrayFragmentsAndLengthValidation(t *testing.T) {
	var v struct {
		Data [256]byte `tlv8:"1"`
	}
	for i := range v.Data {
		v.Data[i] = byte(i)
	}
	wire, err := Marshal(v)
	require.NoError(t, err)
	want := append([]byte{1, 255}, v.Data[:255]...)
	want = append(want, 1, 1, 255)
	require.Equal(t, want, wire)
	var got struct {
		Data [256]byte `tlv8:"1"`
	}
	require.NoError(t, Unmarshal(wire, &got))
	require.Equal(t, v, got)
	for _, wire := range [][]byte{{1, 1, 0}, {1, 3, 0, 0, 0}} {
		var uuid struct {
			Data [2]byte `tlv8:"1"`
		}
		require.NotPanics(t, func() { require.Error(t, Unmarshal(wire, &uuid)) })
	}
}
