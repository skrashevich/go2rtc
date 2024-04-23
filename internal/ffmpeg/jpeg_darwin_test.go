//go:build darwin
// +build darwin

package ffmpeg

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseQuery(t *testing.T) {
	args := parseQuery(nil)
	assert.Equal(t, `ffmpeg -hide_banner -i - -c:v mjpeg -f mjpeg -`, args.String())

	query, err := url.ParseQuery("h=480")
	assert.Nil(t, err)
	args = parseQuery(query)
	assert.Equal(t, `ffmpeg -hide_banner -i - -c:v mjpeg -vf "scale=-1:480" -f mjpeg -`, args.String())

	query, err = url.ParseQuery("hw=vaapi")
	assert.Nil(t, err)
	args = parseQuery(query)
	assert.Equal(t, `ffmpeg -hide_banner -hwaccel vaapi -hwaccel_output_format vaapi -hwaccel_flags allow_profile_mismatch -i - -c:v mjpeg_vaapi -vf "format=vaapi|nv12,hwupload" -f mjpeg -`, args.String())
}
