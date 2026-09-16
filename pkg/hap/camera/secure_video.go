package camera

import (
	"github.com/AlexxIT/go2rtc/pkg/hap"
	"github.com/AlexxIT/go2rtc/pkg/hap/tlv8"
)

// Wire definitions from Apple's HKSV Open Source Compatibility Guide,
// Developer Preview 2026-06-03, sections 3.1 and 4.3-4.5:
// https://developer.apple.com/download/files/HomeKit-Secure-Video-Open-Source-Compatibility-Guide.pdf
// These definitions alone do not implement a Secure Video accessory.
const (
	TypeCameraCapabilitiesService = "8010"
	TypeCameraCapabilities        = "8011"
	TypeSupportedVideoStreamTiers = "8043"
	TypeSupportedAudioStreamTiers = "8044"
	TypeSensorUUID                = "805B"

	SecureVideoPreviewVersion = "17.99"

	// The new stream-tier codec enumeration differs from legacy HAP's H264=0.
	StreamTierCodecAVC  = 1
	StreamTierCodecHEVC = 2

	VideoQualityHighest = 1 // 4K only when a simultaneous 2K High tier exists
	VideoQualityHigh    = 2
	VideoQualityMedium  = 3
	VideoQualityLow     = 4

	StreamTierSampleRate48Khz = 4 // Opus transmission clock, not capture rate
	StreamTierBitDepth16      = 2

	SensorTypePrimary = 1
	SensorIntentMain  = 1
)

type SupportedVideoStreamTiers struct {
	Codec       uint8             `tlv8:"1"`
	PayloadType uint8             `tlv8:"2"`
	Tiers       []VideoStreamTier `tlv8:"3"`
}

type VideoStreamTier struct {
	Identifier     uint32 `tlv8:"1"`
	Quality        uint8  `tlv8:"2"`
	AverageBitrate uint32 `tlv8:"3"` // kbps
	Width          uint16 `tlv8:"4"`
	Height         uint16 `tlv8:"5"`
	FrameRate      uint8  `tlv8:"6"`
}

type SupportedAudioStreamTiers struct {
	Codec       uint8             `tlv8:"1"` // AudioCodecTypeOpus
	PayloadType uint8             `tlv8:"2"`
	Tiers       []AudioStreamTier `tlv8:"3"` // exactly one in the preview spec
}

type AudioStreamTier struct {
	Identifier     uint32 `tlv8:"1"`
	AverageBitrate uint32 `tlv8:"2"` // bits/s, unlike video tiers
	SampleRate     uint8  `tlv8:"3"`
	BitDepth       uint8  `tlv8:"4"`
	PacketTime     uint8  `tlv8:"5"` // 20 ms
	Channels       uint8  `tlv8:"6"` // mono
}

type CameraCapabilities struct {
	Version uint8         `tlv8:"1"`
	Sensors CameraSensors `tlv8:"2"`
}

type CameraSensors struct {
	Sensors []CameraSensorConfiguration `tlv8:"1"`
}

type CameraSensorConfiguration struct {
	Dimensions   SensorDimensions                `tlv8:"1"`
	UUID         [16]byte                        `tlv8:"2"`
	Type         uint8                           `tlv8:"3"`
	Intent       uint8                           `tlv8:"4"`
	VideoStreams []CameraVideoStreamCapabilities `tlv8:"5"`
}

type SensorDimensions struct {
	Width  uint16 `tlv8:"1"`
	Height uint16 `tlv8:"2"`
}

type CameraVideoStreamCapabilities struct {
	Identifier     [16]byte `tlv8:"1"`
	Quality        uint8    `tlv8:"2"`
	Width          uint16   `tlv8:"3"`
	Height         uint16   `tlv8:"4"`
	FrameRate      uint8    `tlv8:"5"`
	AverageBitrate uint32   `tlv8:"6"` // kbps
	PeakBitrate    uint32   `tlv8:"7"` // kbps
}

// ServiceCameraCapabilities encodes caller-supplied sensor metadata. Sensor and
// configuration UUIDs must be stable; no profiles are inferred or fabricated.
// Do not attach this service to a legacy accessory: its presence opts the hub
// into the new protocol, whose transports must be implemented first.
func ServiceCameraCapabilities(capabilities CameraCapabilities) (*hap.Service, error) {
	value, err := tlv8.MarshalBase64(capabilities)
	if err != nil {
		return nil, err
	}
	return &hap.Service{
		Type: TypeCameraCapabilitiesService,
		Characters: []*hap.Character{
			{Type: "37", Format: hap.FormatString, Value: SecureVideoPreviewVersion, Perms: hap.PR},
			{Type: TypeCameraCapabilities, Format: hap.FormatTLV8, Value: value, Perms: hap.PR},
		},
	}, nil
}
