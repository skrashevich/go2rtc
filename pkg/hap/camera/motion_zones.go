package camera

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/AlexxIT/go2rtc/pkg/hap"
	"github.com/AlexxIT/go2rtc/pkg/hap/tlv8"
)

const (
	TypeCameraMotionZones = "8021"
	TypeCameraZones       = "8022"
	CameraZonesVersion    = 2
	ZoneMethodNormal      = 1 // union of polygon interiors
	ZoneMethodInverted    = 2 // intersection of polygon exteriors
)

// CameraZones describes the version 2 payload in section 4.14 of the HKSV
// compatibility guide. Each zone group has its own application method.
type CameraZones struct {
	Version uint8
	Zones   []CameraZoneData
}

type CameraZoneData struct {
	Method   uint8
	Polygons []CameraZonePolygon
}

type CameraZonePolygon struct {
	Identifier [16]byte
	Vertices   []ZonePoint
}

type ZonePoint struct {
	X, Y uint16
}

type cameraZonesWire struct {
	Version uint8                `tlv8:"1"`
	Zones   []cameraZoneDataWire `tlv8:"2"`
}

type cameraZoneDataWire struct {
	Method   uint8                   `tlv8:"1"`
	Polygons []cameraZonePolygonWire `tlv8:"3"`
}

type cameraZonePolygonWire struct {
	Identifier [16]byte `tlv8:"1"`
	// tlv8 uses strings for opaque data; []byte denotes repeated uint8 TLVs.
	Vertices string `tlv8:"3"`
}

func MarshalCameraZones(zones CameraZones, dimensions SensorDimensions) ([]byte, error) {
	if err := validateCameraZones(zones, dimensions); err != nil {
		return nil, err
	}
	wire := cameraZonesWire{Version: zones.Version}
	for _, zone := range zones.Zones {
		group := cameraZoneDataWire{Method: zone.Method}
		for _, polygon := range zone.Polygons {
			vertices := make([]byte, 0, len(polygon.Vertices)*4)
			for _, point := range polygon.Vertices {
				vertices = binary.LittleEndian.AppendUint16(vertices, point.X)
				vertices = binary.LittleEndian.AppendUint16(vertices, point.Y)
			}
			group.Polygons = append(group.Polygons, cameraZonePolygonWire{polygon.Identifier, string(vertices)})
		}
		wire.Zones = append(wire.Zones, group)
	}
	return tlv8.Marshal(wire)
}

// ParseCameraZones validates a complete update before returning it. An empty
// value clears zones; callers must only replace stored state after success.
// This decodes geometry, it does not apply it to a motion detector.
func ParseCameraZones(data []byte, dimensions SensorDimensions) (CameraZones, error) {
	if dimensions.Width == 0 || dimensions.Height == 0 {
		return CameraZones{}, errors.New("camera: invalid sensor dimensions")
	}
	if len(data) == 0 {
		return CameraZones{Version: CameraZonesVersion}, nil
	}
	var wire cameraZonesWire
	if err := tlv8.Unmarshal(data, &wire); err != nil {
		return CameraZones{}, err
	}
	zones := CameraZones{Version: wire.Version}
	for _, group := range wire.Zones {
		zone := CameraZoneData{Method: group.Method}
		for _, raw := range group.Polygons {
			if len(raw.Vertices)%4 != 0 {
				return CameraZones{}, errors.New("camera: incomplete zone coordinate pair")
			}
			polygon := CameraZonePolygon{Identifier: raw.Identifier}
			vertices := []byte(raw.Vertices)
			for i := 0; i < len(vertices); i += 4 {
				polygon.Vertices = append(polygon.Vertices, ZonePoint{
					X: binary.LittleEndian.Uint16(vertices[i:]),
					Y: binary.LittleEndian.Uint16(vertices[i+2:]),
				})
			}
			zone.Polygons = append(zone.Polygons, polygon)
		}
		zones.Zones = append(zones.Zones, zone)
	}
	if err := validateCameraZones(zones, dimensions); err != nil {
		return CameraZones{}, err
	}
	return zones, nil
}

func validateCameraZones(zones CameraZones, dimensions SensorDimensions) error {
	if zones.Version != CameraZonesVersion {
		return fmt.Errorf("camera: unsupported zone version %d", zones.Version)
	}
	if dimensions.Width == 0 || dimensions.Height == 0 {
		return errors.New("camera: invalid sensor dimensions")
	}
	ids := map[[16]byte]bool{}
	for _, zone := range zones.Zones {
		if zone.Method != ZoneMethodNormal && zone.Method != ZoneMethodInverted {
			return fmt.Errorf("camera: invalid zone method %d", zone.Method)
		}
		for _, polygon := range zone.Polygons {
			if polygon.Identifier == [16]byte{} || ids[polygon.Identifier] {
				return errors.New("camera: missing or duplicate zone UUID")
			}
			ids[polygon.Identifier] = true
			if err := validateZonePolygon(polygon.Vertices, dimensions); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateZonePolygon(points []ZonePoint, dimensions SensorDimensions) error {
	if len(points) < 3 {
		return errors.New("camera: zone polygon needs at least three vertices")
	}
	seen := map[ZonePoint]bool{}
	var area int64
	for i, a := range points {
		if a.X > dimensions.Width || a.Y > dimensions.Height {
			return errors.New("camera: zone vertex outside sensor bounds")
		}
		if seen[a] {
			return errors.New("camera: duplicate zone vertex")
		}
		seen[a] = true
		b, c := points[(i+1)%len(points)], points[(i+2)%len(points)]
		area += int64(a.X)*int64(b.Y) - int64(b.X)*int64(a.Y)
		// Adjacent edges may be collinear, but must not double back.
		dot := (int64(b.X)-int64(a.X))*(int64(c.X)-int64(b.X)) +
			(int64(b.Y)-int64(a.Y))*(int64(c.Y)-int64(b.Y))
		if zoneCross(a, b, c) == 0 && dot < 0 {
			return errors.New("camera: overlapping zone edges")
		}
		for j := i + 2; j < len(points); j++ {
			if i == 0 && j == len(points)-1 {
				continue // closing edge is adjacent
			}
			if zoneEdgesIntersect(a, b, points[j], points[(j+1)%len(points)]) {
				return errors.New("camera: self-intersecting zone polygon")
			}
		}
	}
	if area == 0 {
		return errors.New("camera: zone polygon has zero area")
	}
	return nil
}

func zoneCross(a, b, c ZonePoint) int64 {
	return (int64(b.X)-int64(a.X))*(int64(c.Y)-int64(a.Y)) -
		(int64(b.Y)-int64(a.Y))*(int64(c.X)-int64(a.X))
}

func zoneEdgesIntersect(a, b, c, d ZonePoint) bool {
	abc, abd, cda, cdb := zoneCross(a, b, c), zoneCross(a, b, d), zoneCross(c, d, a), zoneCross(c, d, b)
	if ((abc < 0 && abd > 0) || (abc > 0 && abd < 0)) &&
		((cda < 0 && cdb > 0) || (cda > 0 && cdb < 0)) {
		return true
	}
	onSegment := func(p, a, b ZonePoint) bool {
		return p.X >= min(a.X, b.X) && p.X <= max(a.X, b.X) && p.Y >= min(a.Y, b.Y) && p.Y <= max(a.Y, b.Y)
	}
	return abc == 0 && onSegment(c, a, b) || abd == 0 && onSegment(d, a, b) ||
		cda == 0 && onSegment(a, c, d) || cdb == 0 && onSegment(b, c, d)
}

// ServiceCameraMotionZones constructs the versioned metadata for a future
// Secure Video accessory. A nil zones value represents cleared configuration.
func ServiceCameraMotionZones(zones *CameraZones, dimensions SensorDimensions, active bool) (*hap.Service, error) {
	var value string
	if zones != nil {
		data, err := MarshalCameraZones(*zones, dimensions)
		if err != nil {
			return nil, err
		}
		value = base64.StdEncoding.EncodeToString(data)
	}
	var enabled uint8
	if active {
		enabled = 1
	}
	return &hap.Service{
		Type: TypeCameraMotionZones,
		Characters: []*hap.Character{
			{Type: "37", Format: hap.FormatString, Value: SecureVideoPreviewVersion, Perms: hap.PR},
			{Type: "B0", Format: hap.FormatUInt8, Value: enabled, Perms: hap.EVPRPW},
			{Type: TypeCameraZones, Format: hap.FormatTLV8, Value: value, Perms: hap.PRPW},
		},
	}, nil
}
