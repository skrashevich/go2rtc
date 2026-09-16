package homekit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlexxIT/go2rtc/internal/app"
	"github.com/AlexxIT/go2rtc/pkg/hksv"
	"github.com/AlexxIT/go2rtc/pkg/yaml"
	"github.com/stretchr/testify/require"
)

func TestOperatingStateSurvivesRestart(t *testing.T) {
	oldPath, oldReadOnly := app.ConfigPath, app.ConfigReadOnly
	t.Cleanup(func() { app.ConfigPath, app.ConfigReadOnly = oldPath, oldReadOnly })
	app.ConfigPath, app.ConfigReadOnly = filepath.Join(t.TempDir(), "go2rtc.yaml"), false
	require.NoError(t, os.WriteFile(app.ConfigPath, []byte("homekit:\n  camera:\n    hksv: true\n    pairings: [keep-me]\n"), 0600))
	store := &go2rtcPairingStore{}
	s, err := hksv.NewServer(hksv.Config{StreamName: "camera", HKSV: true, StateStore: store})
	require.NoError(t, err)
	active := s.Accessory().GetCharacter("21B")
	require.Zero(t, s.SetCharacteristicStatus(nil, 1, active.IID, false))
	data, err := os.ReadFile(app.ConfigPath)
	require.NoError(t, err)
	var cfg struct {
		HomeKit map[string]struct {
			State    *hksv.OperatingState `yaml:"operating_state"`
			Pairings []string             `yaml:"pairings"`
		} `yaml:"homekit"`
	}
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	require.Equal(t, []string{"keep-me"}, cfg.HomeKit["camera"].Pairings)
	require.NotNil(t, cfg.HomeKit["camera"].State)
	restarted, err := hksv.NewServer(hksv.Config{StreamName: "camera", HKSV: true, OperatingState: cfg.HomeKit["camera"].State})
	require.NoError(t, err)
	require.Equal(t, false, restarted.Accessory().GetCharacter("21B").GetValue())
	require.Empty(t, restarted.GetImage(nil, 320, 240))
	require.Equal(t, active.IID, restarted.Accessory().GetCharacter("21B").IID)
}
