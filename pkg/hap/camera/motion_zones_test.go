package camera

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/AlexxIT/go2rtc/pkg/hap"
	"github.com/stretchr/testify/require"
)

func testCameraZones() CameraZones {
	return CameraZones{Version: 2, Zones: []CameraZoneData{{Method: ZoneMethodNormal, Polygons: []CameraZonePolygon{{
		Identifier: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Vertices:   []ZonePoint{{0, 0}, {640, 0}, {0, 480}},
	}}}}}
}

func TestCameraZonesWire(t *testing.T) {
	dimensions := SensorDimensions{Width: 640, Height: 480}
	// Version=2; zone T=2,L=37; method=1; polygon T=3,L=32;
	// UUID then three raw (X,Y) little-endian pairs, not repeated TLV integers.
	want, err := hex.DecodeString("0101020225010101032001100102030405060708090a0b0c0d0e0f10030c00000000800200000000e001")
	require.NoError(t, err)
	zones := testCameraZones()
	got, err := MarshalCameraZones(zones, dimensions)
	require.NoError(t, err)
	require.Equal(t, want, got)
	decoded, err := ParseCameraZones(want, dimensions)
	require.NoError(t, err)
	require.Equal(t, zones, decoded)
}

func TestCameraZonesListsAndFragments(t *testing.T) {
	zones := testCameraZones()
	for i := 2; i <= 12; i++ {
		p := zones.Zones[0].Polygons[0]
		p.Identifier[0] = byte(i)
		zones.Zones[0].Polygons = append(zones.Zones[0].Polygons, p)
	}
	zones.Zones = append(zones.Zones, CameraZoneData{Method: ZoneMethodInverted, Polygons: []CameraZonePolygon{{
		Identifier: [16]byte{99}, Vertices: []ZonePoint{{0, 0}, {10, 0}, {0, 10}},
	}}})
	dimensions := SensorDimensions{Width: 640, Height: 480}
	data, err := MarshalCameraZones(zones, dimensions)
	require.NoError(t, err)
	require.Greater(t, len(data), 255)
	got, err := ParseCameraZones(data, dimensions)
	require.NoError(t, err)
	require.Equal(t, zones, got)

	// Exercise fragmentation inside the raw vertex buffer too.
	zones = testCameraZones()
	vertices := []ZonePoint{{0, 0}}
	for x := uint16(1); x <= 70; x++ {
		vertices = append(vertices, ZonePoint{x, 0})
	}
	vertices = append(vertices, ZonePoint{70, 10}, ZonePoint{0, 10})
	zones.Zones[0].Polygons[0].Vertices = vertices
	data, err = MarshalCameraZones(zones, dimensions)
	require.NoError(t, err)
	got, err = ParseCameraZones(data, dimensions)
	require.NoError(t, err)
	require.Equal(t, zones, got)
}

func TestCameraZonesRejectInvalidGeometry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		points []ZonePoint
	}{
		{"too few", []ZonePoint{{0, 0}, {10, 10}}},
		{"outside", []ZonePoint{{0, 0}, {641, 0}, {0, 480}}},
		{"line", []ZonePoint{{0, 0}, {10, 10}, {20, 20}}},
		{"duplicate", []ZonePoint{{0, 0}, {10, 0}, {10, 0}, {0, 10}}},
		{"crossing", []ZonePoint{{0, 0}, {20, 20}, {0, 20}, {20, 0}}},
		{"unequal lobes", []ZonePoint{{0, 0}, {30, 20}, {0, 20}, {20, 0}}},
		{"backtracking", []ZonePoint{{0, 0}, {20, 0}, {10, 0}, {10, 20}, {0, 20}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			zones := testCameraZones()
			zones.Zones[0].Polygons[0].Vertices = tc.points
			_, err := MarshalCameraZones(zones, SensorDimensions{640, 480})
			require.Error(t, err)
		})
	}
}

func TestCameraZonesValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*CameraZones)
	}{
		{"version", func(z *CameraZones) { z.Version = 1 }},
		{"method", func(z *CameraZones) { z.Zones[0].Method = 9 }},
		{"missing UUID", func(z *CameraZones) { z.Zones[0].Polygons[0].Identifier = [16]byte{} }},
		{"duplicate UUID", func(z *CameraZones) { z.Zones[0].Polygons = append(z.Zones[0].Polygons, z.Zones[0].Polygons[0]) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			zones := testCameraZones()
			tc.mutate(&zones)
			_, err := MarshalCameraZones(zones, SensorDimensions{640, 480})
			require.Error(t, err)
		})
	}
	_, err := MarshalCameraZones(testCameraZones(), SensorDimensions{})
	require.Error(t, err)
	// Large coordinates must not overflow int32 orientation calculations.
	zones := testCameraZones()
	zones.Zones[0].Polygons[0].Vertices = []ZonePoint{{0, 0}, {65535, 0}, {65535, 65535}, {0, 65535}}
	_, err = MarshalCameraZones(zones, SensorDimensions{65535, 65535})
	require.NoError(t, err)
}

func TestCameraZonesAcceptConcaveAndClockwisePolygons(t *testing.T) {
	for _, points := range [][]ZonePoint{
		{{0, 0}, {20, 0}, {20, 10}, {10, 10}, {10, 20}, {0, 20}},
		{{0, 20}, {10, 20}, {10, 10}, {20, 10}, {20, 0}, {0, 0}},
	} {
		zones := testCameraZones()
		zones.Zones[0].Polygons[0].Vertices = points
		data, err := MarshalCameraZones(zones, SensorDimensions{640, 480})
		require.NoError(t, err)
		got, err := ParseCameraZones(data, SensorDimensions{640, 480})
		require.NoError(t, err)
		require.Equal(t, zones, got)
	}
}

func FuzzParseCameraZones(f *testing.F) {
	data, _ := hex.DecodeString("0101020225010101032001100102030405060708090a0b0c0d0e0f10030c00000000800200000000e001")
	f.Add(data)
	f.Add([]byte{})
	f.Add([]byte{1, 1, 2})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			t.Skip() // bound geometric validation work in the fuzz corpus
		}
		zones, err := ParseCameraZones(data, SensorDimensions{640, 480})
		if err != nil {
			return
		}
		encoded, err := MarshalCameraZones(zones, SensorDimensions{640, 480})
		require.NoError(t, err)
		decoded, err := ParseCameraZones(encoded, SensorDimensions{640, 480})
		require.NoError(t, err)
		require.Equal(t, zones, decoded)
	})
}

func TestCameraZonesMalformedWire(t *testing.T) {
	for _, wire := range []string{
		"010101",           // unknown version
		"02",               // truncated header
		"0101020204",       // truncated value
		"0101020203010109", // invalid method
		"0101020224010101031f01100102030405060708090a0b0c0d0e0f10030b00000000800200000000e0",   // partial coordinate
		"0101020225010101032001100102030405060708090a0b0c0d0e0f10030c00000000810200000000e001", // outside frame
	} {
		data, err := hex.DecodeString(wire)
		require.NoError(t, err)
		_, err = ParseCameraZones(data, SensorDimensions{640, 480})
		require.Error(t, err, wire)
	}
}

func TestMotionZonesServiceAndClear(t *testing.T) {
	zones := testCameraZones()
	service, err := ServiceCameraMotionZones(&zones, SensorDimensions{640, 480}, true)
	require.NoError(t, err)
	acc := &hap.Accessory{AID: 1, Services: []*hap.Service{service}}
	acc.InitIID()
	require.Equal(t, "8021", service.Type)
	require.Len(t, service.Characters, 3)
	require.Equal(t, "17.99", service.GetCharacter("37").Value)
	require.Equal(t, uint8(1), service.GetCharacter("B0").Value)
	char := service.GetCharacter("8022")
	require.Equal(t, "tlv8", char.Format)
	require.ElementsMatch(t, []string{"pr", "pw"}, char.Perms)
	b, err := base64.StdEncoding.DecodeString(char.Value.(string))
	require.NoError(t, err)
	got, err := ParseCameraZones(b, SensorDimensions{640, 480})
	require.NoError(t, err)
	require.Equal(t, zones, got)

	service, err = ServiceCameraMotionZones(nil, SensorDimensions{640, 480}, false)
	require.NoError(t, err)
	require.Equal(t, "", service.GetCharacter("8022").Value)
	require.Equal(t, uint8(0), service.GetCharacter("B0").Value)
	cleared, err := ParseCameraZones(nil, SensorDimensions{640, 480})
	require.NoError(t, err)
	require.Equal(t, CameraZones{Version: 2}, cleared)
}
