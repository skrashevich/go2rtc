package camera

import (
	"encoding/json"
	"testing"

	"github.com/AlexxIT/go2rtc/pkg/hap"
	"github.com/stretchr/testify/require"
)

func TestGlobalOperatingModeStreamingGate(t *testing.T) {
	// All four gates independently inhibit streaming. The LED indicator does not.
	for bits := range 32 {
		mode := CameraGlobalOperatingMode{
			CameraActive: bits&1 != 0, StreamingEnabled: bits&2 != 0,
			ManuallyDisabled: bits&4 != 0, IndicatorEnabled: bits&8 != 0,
		}
		streamEnabled := bits&16 != 0
		want := bits == 19 || bits == 27
		require.Equal(t, want, mode.AllowsStreaming(streamEnabled), "flags %05b", bits)
	}
}

func TestGlobalOperatingModeService(t *testing.T) {
	mode := CameraGlobalOperatingMode{CameraActive: true, StreamingEnabled: false, IndicatorEnabled: true, ManuallyDisabled: true}
	service := ServiceCameraGlobalOperatingMode(mode)
	acc := &hap.Accessory{AID: 1, Services: []*hap.Service{service}}
	acc.InitIID()
	require.Equal(t, "8032", service.Type)
	require.Nil(t, acc.GetService("110"))
	require.Len(t, service.Characters, 4)
	for _, tc := range []struct {
		typ, format string
		value       any
		perms       []string
	}{
		{"21B", "uint8", uint8(1), []string{"ev", "pr", "pw"}},
		{"8041", "bool", false, []string{"ev", "pr", "pw", "tw"}},
		{"21D", "bool", true, []string{"ev", "pr", "pw", "tw"}},
		{"227", "bool", true, []string{"ev", "pr"}},
	} {
		char := service.GetCharacter(tc.typ)
		require.NotNil(t, char)
		require.Equal(t, tc.format, char.Format)
		require.Equal(t, tc.value, char.Value)
		require.ElementsMatch(t, tc.perms, char.Perms)
	}
	// A false value must be present in the HAP response (not omitted).
	b, err := json.Marshal(service.GetCharacter("8041"))
	require.NoError(t, err)
	var wire map[string]any
	require.NoError(t, json.Unmarshal(b, &wire))
	require.Equal(t, false, wire["value"])
}
