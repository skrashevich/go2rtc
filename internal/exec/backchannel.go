package exec

import (
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/opus"
	"github.com/AlexxIT/go2rtc/pkg/pcm"
	"github.com/AlexxIT/go2rtc/pkg/shell"
)

// Keep Opus conversion at the exec boundary: pkg/opus itself uses pkg/pcm.
type backchannel struct {
	*pcm.Backchannel
	audio opus.Consumer
}

func newBackchannel(cmd *shell.Command, format string) (core.Producer, error) {
	prod, err := pcm.NewBackchannel(cmd, format)
	if err != nil {
		return nil, err
	}
	return &backchannel{Backchannel: prod.(*pcm.Backchannel)}, nil
}

func (c *backchannel) GetMedias() []*core.Media {
	return c.audio.GetMedias(c.Medias)
}

func (c *backchannel) AddTrack(media *core.Media, codec *core.Codec, track *core.Receiver) error {
	return c.audio.AddTrack(media, codec, track, c.Backchannel.AddTrack)
}

func (c *backchannel) Stop() error {
	defer c.audio.Close()
	return c.Backchannel.Stop()
}
