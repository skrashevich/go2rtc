package ffmpeg

import (
	"bytes"
	"os/exec"
	"strings"
)

const (
	FFmpeg50 = "59. 16"
	FFmpeg51 = "59. 27"
	FFmpeg60 = "60.  3"
	FFmpeg61 = "60. 16"
	FFmpeg70 = "61.  1"
)

func Version() (string, error) {
	cmd := exec.Command(defaults["ffmpeg"], "-version")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	firstLine := strings.Split(out.String(), "\n")[0]
	versionInfo := strings.Fields(firstLine)[2]
	return versionInfo, nil
}
