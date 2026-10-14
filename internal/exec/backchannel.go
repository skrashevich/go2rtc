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
}

func newBackchannel(cmd *shell.Command, format string) (core.Producer, error) {
	prod, err := pcm.NewBackchannel(cmd, format)
	if err != nil {
		return nil, err
	}
	return &backchannel{Backchannel: prod.(*pcm.Backchannel)}, nil
}

func (c *backchannel) GetMedias() []*core.Media {
	return opus.WithTranscoding(c.Medias)
}

func (c *backchannel) AddTrack(media *core.Media, codec *core.Codec, track *core.Receiver) error {
	return opus.AddTrack(c.Medias, media, codec, track, c.Backchannel.AddTrack)
}
