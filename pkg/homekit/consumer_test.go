package homekit

import (
	"net"
	"testing"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/hap/camera"
	"github.com/AlexxIT/go2rtc/pkg/srtp"
	"github.com/pion/rtp"
	"github.com/stretchr/testify/require"
)

type testControlConn struct{ net.Conn }

func (c testControlConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}

func TestLiveStopKeepsControlConnectionAndDropsQueuedMedia(t *testing.T) {
	control, controller := net.Pipe()
	t.Cleanup(func() { control.Close(); controller.Close() })
	listener, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })
	server := srtp.NewServer("127.0.0.1:0")
	consumer := NewConsumer(testControlConn{control}, server)
	t.Cleanup(func() { _ = consumer.Stop() })
	crypto := camera.SRTPCryptoSuite{MasterKey: "0123456789abcdef", MasterSalt: "0123456789abcd"}
	consumer.SetOffer(&camera.SetupEndpointsRequest{
		SessionID: "test-session", Address: camera.Address{IPAddr: "127.0.0.1", VideoRTPPort: uint16(listener.LocalAddr().(*net.UDPAddr).Port), AudioRTPPort: uint16(listener.LocalAddr().(*net.UDPAddr).Port)},
		VideoCrypto: crypto, AudioCrypto: crypto,
	})
	answer := consumer.GetAnswer()
	require.Equal(t, answer, consumer.GetAnswer(), "reading endpoints must not replace negotiated keys")
	require.True(t, consumer.SetConfig(&camera.SelectedStreamConfiguration{
		Control:    camera.SessionControl{SessionID: "test-session"},
		VideoCodec: camera.VideoCodecConfiguration{RTPParams: []camera.RTPParams{{SSRC: 101, PayloadType: 99, RTCPInterval: 60}}},
		AudioCodec: camera.AudioCodecConfiguration{RTPParams: []camera.RTPParams{{SSRC: 102, PayloadType: 110, RTCPInterval: 60}}, CodecParams: []camera.AudioCodecParameters{{RTPTime: []uint8{20}}}},
	}))
	require.NotNil(t, server.GetSession(101))
	require.NotNil(t, server.GetSession(102))
	media := consumer.Medias[1]
	codec := &core.Codec{Name: core.CodecOpus, ClockRate: 48000, Channels: 1}
	require.NoError(t, consumer.AddTrack(media, codec, core.NewReceiver(media, codec)))
	sender := consumer.Senders[0]
	// Use the real packetization/encryption callback that also drains queued media.
	sender.Handler(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 1}, Payload: []byte{0xF8, 0xFF, 0xFE}})
	require.NoError(t, listener.SetReadDeadline(time.Now().Add(time.Second)))
	packet := make([]byte, 1500)
	n, _, err := listener.ReadFrom(packet)
	require.NoError(t, err)
	require.Greater(t, n, 12)
	require.Equal(t, byte(110), packet[1]&0x7f)
	require.NoError(t, consumer.Stop())
	require.Nil(t, server.GetSession(101))
	require.Nil(t, server.GetSession(102))
	sender.Handler(&rtp.Packet{Header: rtp.Header{Version: 2, SequenceNumber: 2}, Payload: []byte{0xF8, 0xFF, 0xFE}})
	require.NoError(t, listener.SetReadDeadline(time.Now().Add(50*time.Millisecond)))
	_, _, err = listener.ReadFrom(packet)
	require.Error(t, err)
	require.True(t, err.(net.Error).Timeout(), "stopped sender leaked a packet")
	require.NoError(t, controller.SetReadDeadline(time.Now().Add(time.Second)))
	sent := make(chan error, 1)
	go func() { _, err := control.Write([]byte("ok")); sent <- err }()
	n, err = controller.Read(packet)
	require.NoError(t, err)
	require.Equal(t, "ok", string(packet[:n]))
	require.NoError(t, <-sent)
}
