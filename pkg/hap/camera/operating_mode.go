package camera

import "github.com/AlexxIT/go2rtc/pkg/hap"

const (
	TypeCameraGlobalOperatingMode    = "8032"
	TypeStreamingEnabled             = "8041"
	TypeHomeKitCameraActive          = "21B"
	TypeCameraOperatingModeIndicator = "21D"
	TypeManuallyDisabled             = "227"
)

// CameraGlobalOperatingMode is a snapshot of the accessory-wide controls from
// sections 3.2 and 3.6 of the HKSV compatibility guide. Its zero value disables
// streaming. The indicator controls the camera's LED, not access to media.
type CameraGlobalOperatingMode struct {
	CameraActive     bool
	StreamingEnabled bool
	IndicatorEnabled bool
	ManuallyDisabled bool
}

// AllowsStreaming combines the global state with a particular RTP/WebRTC
// service's Streaming Enabled flag. Future stream handlers must check this
// gate on start and stop active sessions when a controlling flag is disabled.
func (m CameraGlobalOperatingMode) AllowsStreaming(streamEnabled bool) bool {
	return m.CameraActive && m.StreamingEnabled && !m.ManuallyDisabled && streamEnabled
}

// ServiceCameraGlobalOperatingMode builds metadata only; it is not attached to
// legacy cameras. Timed writes and the Streaming Enabled admin-only rule must
// be enforced by the future Secure Video request handler before publishing it.
func ServiceCameraGlobalOperatingMode(m CameraGlobalOperatingMode) *hap.Service {
	var active uint8
	if m.CameraActive {
		active = 1
	}
	return &hap.Service{
		Type: TypeCameraGlobalOperatingMode,
		Characters: []*hap.Character{
			{Type: TypeHomeKitCameraActive, Format: hap.FormatUInt8, Value: active, Perms: hap.EVPRPW},
			{Type: TypeStreamingEnabled, Format: hap.FormatBool, Value: m.StreamingEnabled, Perms: []string{"ev", "pr", "pw", "tw"}},
			{Type: TypeCameraOperatingModeIndicator, Format: hap.FormatBool, Value: m.IndicatorEnabled, Perms: []string{"ev", "pr", "pw", "tw"}},
			{Type: TypeManuallyDisabled, Format: hap.FormatBool, Value: m.ManuallyDisabled, Perms: hap.EVPR},
		},
	}
}
