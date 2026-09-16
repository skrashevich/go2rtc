package homekit

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/h264"
	"github.com/AlexxIT/go2rtc/pkg/hap/camera"
	"github.com/AlexxIT/go2rtc/pkg/opus"
	"github.com/AlexxIT/go2rtc/pkg/srtp"
	"github.com/pion/rtp"
)

type Consumer struct {
	core.Connection
	conn net.Conn
	srtp *srtp.Server

	deadline   *time.Timer
	mu         sync.Mutex
	stopped    bool
	configured bool
	done       chan struct{}

	sessionID    string
	videoSession *srtp.Session
	audioSession *srtp.Session
	audioRTPTime byte

	backTrack *core.Receiver // backchannel audio (HomeKit viewer → camera)
}

func NewConsumer(conn net.Conn, server *srtp.Server) *Consumer {
	medias := []*core.Media{
		{
			Kind:      core.KindVideo,
			Direction: core.DirectionSendonly,
			Codecs: []*core.Codec{
				{Name: core.CodecH264},
			},
		},
		{
			Kind:      core.KindAudio,
			Direction: core.DirectionSendonly,
			Codecs: []*core.Codec{
				{Name: core.CodecOpus},
			},
		},
		{
			Kind:      core.KindAudio,
			Direction: core.DirectionRecvonly,
			Codecs: []*core.Codec{
				{Name: core.CodecOpus},
			},
		},
	}
	return &Consumer{
		Connection: core.Connection{
			ID:         core.NewID(),
			FormatName: "homekit",
			Protocol:   "rtp",
			RemoteAddr: conn.RemoteAddr().String(),
			Medias:     medias,
		},
		done: make(chan struct{}),
		conn: conn,
		srtp: server,
	}
}

func (c *Consumer) SessionID() string {
	return c.sessionID
}

func (c *Consumer) SetOffer(offer *camera.SetupEndpointsRequest) {
	c.sessionID = offer.SessionID
	c.videoSession = &srtp.Session{
		Remote: &srtp.Endpoint{
			Addr:       offer.Address.IPAddr,
			Port:       offer.Address.VideoRTPPort,
			MasterKey:  []byte(offer.VideoCrypto.MasterKey),
			MasterSalt: []byte(offer.VideoCrypto.MasterSalt),
		},
	}
	c.audioSession = &srtp.Session{
		Remote: &srtp.Endpoint{
			Addr:       offer.Address.IPAddr,
			Port:       offer.Address.AudioRTPPort,
			MasterKey:  []byte(offer.AudioCrypto.MasterKey),
			MasterSalt: []byte(offer.AudioCrypto.MasterSalt),
		},
	}
}

func (c *Consumer) GetAnswer() *camera.SetupEndpointsResponse {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.videoSession.Local == nil {
		c.videoSession.Local = c.srtpEndpoint()
		c.audioSession.Local = c.srtpEndpoint()
	}

	return &camera.SetupEndpointsResponse{
		SessionID: c.sessionID,
		Status:    camera.StreamingStatusAvailable,
		Address: camera.Address{
			IPAddr:       c.videoSession.Local.Addr,
			VideoRTPPort: c.videoSession.Local.Port,
			AudioRTPPort: c.audioSession.Local.Port,
		},
		VideoCrypto: camera.SRTPCryptoSuite{
			MasterKey:  string(c.videoSession.Local.MasterKey),
			MasterSalt: string(c.videoSession.Local.MasterSalt),
		},
		AudioCrypto: camera.SRTPCryptoSuite{
			MasterKey:  string(c.audioSession.Local.MasterKey),
			MasterSalt: string(c.audioSession.Local.MasterSalt),
		},
		VideoSSRC: c.videoSession.Local.SSRC,
		AudioSSRC: c.audioSession.Local.SSRC,
	}
}

func (c *Consumer) SetConfig(conf *camera.SelectedStreamConfiguration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped || c.configured || c.sessionID != conf.Control.SessionID || len(conf.VideoCodec.RTPParams) == 0 || len(conf.AudioCodec.RTPParams) == 0 || len(conf.AudioCodec.CodecParams) == 0 || len(conf.AudioCodec.CodecParams[0].RTPTime) == 0 {
		return false
	}

	c.SDP = fmt.Sprintf("%+v\n%+v", conf.VideoCodec, conf.AudioCodec)

	c.videoSession.Remote.SSRC = conf.VideoCodec.RTPParams[0].SSRC
	c.videoSession.PayloadType = conf.VideoCodec.RTPParams[0].PayloadType
	c.videoSession.RTCPInterval = toDuration(conf.VideoCodec.RTPParams[0].RTCPInterval)

	c.audioSession.Remote.SSRC = conf.AudioCodec.RTPParams[0].SSRC
	c.audioSession.PayloadType = conf.AudioCodec.RTPParams[0].PayloadType
	c.audioSession.RTCPInterval = toDuration(conf.AudioCodec.RTPParams[0].RTCPInterval)
	c.audioRTPTime = conf.AudioCodec.CodecParams[0].RTPTime[0]

	c.srtp.AddSession(c.videoSession)
	c.srtp.AddSession(c.audioSession)
	c.configured = true

	return true
}

func (c *Consumer) GetTrack(media *core.Media, codec *core.Codec) (*core.Receiver, error) {
	if codec.Kind() != core.KindAudio {
		return nil, core.ErrCantGetTrack
	}

	c.backTrack = core.NewReceiver(media, codec)

	c.audioSession.OnReadRTP = func(packet *rtp.Packet) {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.stopped {
			return
		}
		c.backTrack.WriteRTP(packet)
		c.Recv += len(packet.Payload)
	}

	c.Receivers = append(c.Receivers, c.backTrack)
	return c.backTrack, nil
}

func (c *Consumer) Start() error {
	return nil
}

func (c *Consumer) AddTrack(media *core.Media, codec *core.Codec, track *core.Receiver) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped {
		return fmt.Errorf("homekit: stream stopped")
	}
	var session *srtp.Session
	if codec.Kind() == core.KindVideo {
		session = c.videoSession
	} else {
		session = c.audioSession
	}

	sender := core.NewSender(media, track.Codec)

	resetDeadline := c.deadline == nil
	if resetDeadline {
		c.deadline = time.NewTimer(time.Second * 30)
	}
	sender.Handler = func(packet *rtp.Packet) {
		c.mu.Lock()
		defer c.mu.Unlock()
		// Sender.Close drains its queue; discard those packets once stopped.
		if c.stopped {
			return
		}
		if resetDeadline {
			c.deadline.Reset(core.ConnDeadline)
		}
		if n, err := session.WriteRTP(packet); err == nil {
			c.Send += n
		}
	}

	switch codec.Name {
	case core.CodecH264:
		sender.Handler = h264.RTPPay(1378, sender.Handler)
		if track.Codec.IsRTP() {
			sender.Handler = h264.RTPDepay(track.Codec, sender.Handler)
		} else {
			sender.Handler = h264.RepairAVCC(track.Codec, sender.Handler)
		}
	case core.CodecOpus:
		sender.Handler = opus.RepackToHAP(c.audioRTPTime, sender.Handler)
	}

	sender.HandleRTP(track)
	c.Senders = append(c.Senders, sender)
	return nil
}

func (c *Consumer) WriteTo(io.Writer) (int64, error) {
	c.mu.Lock()
	deadline := c.deadline
	c.mu.Unlock()
	if deadline != nil {
		select {
		case <-deadline.C:
		case <-c.done:
		}
	}
	return 0, nil
}

func (c *Consumer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped {
		return nil
	}
	c.stopped = true
	close(c.done)
	if c.deadline != nil {
		c.deadline.Stop()
	}
	if c.configured {
		c.srtp.DelSession(c.videoSession)
		c.srtp.DelSession(c.audioSession)
	}
	// The HAP control connection is shared with camera settings and other streams.
	return c.Connection.Stop()
}

func (c *Consumer) srtpEndpoint() *srtp.Endpoint {
	addr := c.conn.LocalAddr().(*net.TCPAddr)
	return &srtp.Endpoint{
		Addr:       addr.IP.To4().String(),
		Port:       uint16(c.srtp.Port()),
		MasterKey:  []byte(core.RandString(16, 0)),
		MasterSalt: []byte(core.RandString(14, 0)),
		SSRC:       rand.Uint32(),
	}
}

func toDuration(seconds float32) time.Duration {
	return time.Duration(seconds * float32(time.Second))
}
