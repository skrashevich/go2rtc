package rtsp

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeout(t *testing.T) {
	Timeout = time.Millisecond

	ln, err := net.Listen("tcp", "localhost:0")
	assert.Nil(t, err)

	client := NewClient("rtsp://" + ln.Addr().String() + "/stream")
	client.Backchannel = true

	err = client.Dial()
	assert.Nil(t, err)

	err = client.Describe()
	assert.ErrorIs(t, err, os.ErrDeadlineExceeded)
}

func TestMissedControl(t *testing.T) {
	Timeout = time.Millisecond

	ln, err := net.Listen("tcp", "localhost:0")
	assert.Nil(t, err)

	go func() {
		conn, err := ln.Accept()
		assert.Nil(t, err)

		b := make([]byte, 8192)
		for {
			n, err := conn.Read(b)
			assert.Nil(t, err)

			req := string(b[:n])

			switch req[:4] {
			case "DESC":
				_, _ = conn.Write([]byte(`RTSP/1.0 200 OK
Cseq: 1
Content-Length: 495
Content-Type: application/sdp

v=0
o=- 1 1 IN IP4 0.0.0.0
s=go2rtc/1.2.0
c=IN IP4 0.0.0.0
t=0 0
m=audio 0 RTP/AVP 96
a=rtpmap:96 MPEG4-GENERIC/48000/2
a=fmtp:96 profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3; config=119056E500
m=audio 0 RTP/AVP 97
a=rtpmap:97 OPUS/48000/2
a=fmtp:97 sprop-stereo=1
m=video 0 RTP/AVP 98
a=rtpmap:98 H264/90000
a=fmtp:98 packetization-mode=1; sprop-parameter-sets=Z2QAKaw0yAeAIn5cBagICAoAAAfQAAE4gdDAAjhAACOEF3lxoYAEcIAARwgu8uFA,aO48MAA=; profile-level-id=640029
`))

			case "SETU":
				_, _ = conn.Write([]byte(`RTSP/1.0 200 OK
Transport: RTP/AVP/TCP;unicast;interleaved=4-5
Cseq: 3
Session: 1

`))

			default:
				t.Fail()
			}
		}
	}()

	client := NewClient("rtsp://" + ln.Addr().String() + "/stream")
	client.Backchannel = true

	err = client.Dial()
	assert.Nil(t, err)

	err = client.Describe()
	assert.Nil(t, err)
	assert.Len(t, client.Medias, 3)

	ch, err := client.SetupMedia(client.Medias[2])
	assert.Nil(t, err)
	assert.Equal(t, ch, byte(4))
}
