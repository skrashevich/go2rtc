package h265

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeSPS(t *testing.T) {
	s := "QgEBAWAAAAMAAAMAAAMAAAMAmaAAoAgBaH+KrTuiS7/8AAQABbAgApMuADN/mAE="
	b, err := base64.StdEncoding.DecodeString(s)
	assert.Nil(t, err)

	sps := DecodeSPS(b)
	assert.NotNil(t, sps)
	assert.Equal(t, uint16(5120), sps.Width())
	assert.Equal(t, uint16(1440), sps.Height())
}
