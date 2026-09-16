package camera

import (
	"encoding/hex"
	"testing"

	"github.com/AlexxIT/go2rtc/pkg/hap"
	"github.com/AlexxIT/go2rtc/pkg/hap/tlv8"
	"github.com/stretchr/testify/require"
)

func TestSecureVideoTierWireFormat(t *testing.T) {
	// Section 4.3: HEVC=2, PT=100, one 4K High tier at 4.5 Mbps.
	want, err := hex.DecodeString("010102020164031a0104010000000201020304941100000402000f0502700806011e")
	require.NoError(t, err)
	v := SupportedVideoStreamTiers{
		Codec: StreamTierCodecHEVC, PayloadType: 100,
		Tiers: []VideoStreamTier{{Identifier: 1, Quality: VideoQualityHigh, AverageBitrate: 4500, Width: 3840, Height: 2160, FrameRate: 30}},
	}
	got, err := tlv8.Marshal(v)
	require.NoError(t, err)
	require.Equal(t, want, got)
	var decoded SupportedVideoStreamTiers
	require.NoError(t, tlv8.Unmarshal(want, &decoded))
	require.Equal(t, v, decoded)

	// New tier AVC=1 must not accidentally reuse legacy H.264=0.
	v.Codec = StreamTierCodecAVC
	got, err = tlv8.Marshal(v)
	require.NoError(t, err)
	require.Equal(t, byte(1), got[2])
}

func TestSecureAudioTierWireFormat(t *testing.T) {
	// Section 4.4: bitrate is bits/s; Opus transmission clock is 48 kHz.
	want, err := hex.DecodeString("01010302016503180104010000000204007d0000030104040102050114060101")
	require.NoError(t, err)
	v := SupportedAudioStreamTiers{Codec: AudioCodecTypeOpus, PayloadType: 101,
		Tiers: []AudioStreamTier{{Identifier: 1, AverageBitrate: 32000, SampleRate: StreamTierSampleRate48Khz,
			BitDepth: StreamTierBitDepth16, PacketTime: 20, Channels: 1}},
	}
	got, err := tlv8.Marshal(v)
	require.NoError(t, err)
	require.Equal(t, want, got)
	var decoded SupportedAudioStreamTiers
	require.NoError(t, tlv8.Unmarshal(want, &decoded))
	require.Equal(t, v, decoded)
}

func TestCameraCapabilitiesNestedSensors(t *testing.T) {
	// Multiple sensors make nested TLVs cross 255 bytes, as real multi-sensor
	// accessories do. UUID bytes must survive without endian conversion.
	sensor := CameraSensorConfiguration{
		Dimensions: SensorDimensions{Width: 3840, Height: 2160},
		UUID:       [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Type:       SensorTypePrimary, Intent: SensorIntentMain,
		VideoStreams: []CameraVideoStreamCapabilities{
			{Identifier: [16]byte{1}, Quality: VideoQualityHigh, Width: 3840, Height: 2160, FrameRate: 30, AverageBitrate: 4500, PeakBitrate: 5000},
			{Identifier: [16]byte{2}, Quality: VideoQualityMedium, Width: 1920, Height: 1080, FrameRate: 30, AverageBitrate: 1700, PeakBitrate: 1800},
			{Identifier: [16]byte{3}, Quality: VideoQualityLow, Width: 640, Height: 360, FrameRate: 15, AverageBitrate: 180, PeakBitrate: 190},
		},
	}
	second := sensor
	second.UUID[0] = 99
	v := CameraCapabilities{Version: 1, Sensors: CameraSensors{Sensors: []CameraSensorConfiguration{sensor, second}}}
	service, err := ServiceCameraCapabilities(v)
	require.NoError(t, err)
	acc := &hap.Accessory{AID: 1, Services: []*hap.Service{service}}
	require.NotPanics(t, acc.InitIID)
	require.Equal(t, "8010", service.Type)
	require.Nil(t, acc.GetService("110"), "metadata does not add a legacy streaming service")
	require.Equal(t, "17.99", service.GetCharacter("37").Value)
	char := service.GetCharacter("8011")
	require.NotNil(t, char)
	require.Equal(t, []string{"pr"}, char.Perms)
	var decoded CameraCapabilities
	require.NoError(t, char.ReadTLV8(&decoded))
	require.Equal(t, v, decoded)

	// Independently check the nested TLV header: version, sensors, sensor list,
	// then dimensions. A missing CameraSensors wrapper changes these tags.
	wire, err := tlv8.Marshal(CameraCapabilities{Version: 1, Sensors: CameraSensors{Sensors: []CameraSensorConfiguration{
		{Dimensions: SensorDimensions{Width: 1920, Height: 1080}, UUID: [16]byte{1}, Type: SensorTypePrimary, Intent: SensorIntentMain},
	}}})
	require.NoError(t, err)
	require.Equal(t, []byte{1, 1, 1, 2, 36, 1, 34, 1, 8, 1, 2, 0x80, 7, 2, 2, 0x38, 4}, wire[:17])
}
