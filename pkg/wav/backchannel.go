package wav

import (
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/opus"
	"github.com/AlexxIT/go2rtc/pkg/shell"
	"github.com/pion/rtp"
)

type Backchannel struct {
	audio opus.Consumer
	core.Connection
	cmd *shell.Command
}

func NewBackchannel(cmd *shell.Command) (core.Producer, error) {
	medias := []*core.Media{
		{
			Kind:      core.KindAudio,
			Direction: core.DirectionSendonly,
			Codecs: []*core.Codec{
				//{Name: core.CodecPCML},
				{Name: core.CodecPCMA},
				{Name: core.CodecPCMU},
			},
		},
	}

	return &Backchannel{
		Connection: core.Connection{
			ID:         core.NewID(),
			FormatName: "wav",
			Protocol:   "pipe",
			Medias:     medias,
			Transport:  cmd,
		},
		cmd: cmd,
	}, nil
}

func (c *Backchannel) GetTrack(media *core.Media, codec *core.Codec) (*core.Receiver, error) {
	return nil, core.ErrCantGetTrack
}

func (c *Backchannel) AddTrack(media *core.Media, codec *core.Codec, track *core.Receiver) error {
	return c.audio.AddTrack(media, codec, track, c.addTrack)
}

func (c *Backchannel) addTrack(media *core.Media, codec *core.Codec, track *core.Receiver) error {
	wr, err := c.cmd.StdinPipe()
	if err != nil {
		return err
	}

	b := Header(track.Codec)
	if _, err = wr.Write(b); err != nil {
		return err
	}

	sender := core.NewSender(media, track.Codec)
	sender.Handler = func(packet *rtp.Packet) {
		if n, err := wr.Write(packet.Payload); err != nil {
			c.Send += n
		}
	}
	sender.HandleRTP(track)
	c.Senders = append(c.Senders, sender)
	return nil
}

func (c *Backchannel) Start() error {
	return c.cmd.Run()
}

func (c *Backchannel) GetMedias() []*core.Media {
	return c.audio.GetMedias(c.Medias)
}

func (c *Backchannel) Stop() error {
	defer c.audio.Close()
	return c.Connection.Stop()
}
