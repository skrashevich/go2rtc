package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchHost(t *testing.T) {
	v, err := Run(`
let url = "rtsp://user:pass@192.168.1.123/cam/realmonitor?...";
let host = match(url, "//[^/]+")[0][2:];
host
`)
	assert.Nil(t, err)
	assert.Equal(t, "user:pass@192.168.1.123", v)
}
