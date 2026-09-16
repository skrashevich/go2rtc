package hksv

import (
	"maps"
	"net"
	"slices"

	"github.com/AlexxIT/go2rtc/pkg/hap"
	"github.com/AlexxIT/go2rtc/pkg/hap/camera"
	"github.com/AlexxIT/go2rtc/pkg/homekit"
)

// OperatingState contains the controls written by Home. Keep the complete state
// together when persisting it so a restart cannot accidentally enable the camera.
type OperatingState struct {
	CameraActive            bool `yaml:"camera_active" json:"camera_active"`
	StreamingActive         bool `yaml:"streaming_active" json:"streaming_active"`
	RecordingActive         bool `yaml:"recording_active" json:"recording_active"`
	RecordingAudioActive    bool `yaml:"recording_audio_active" json:"recording_audio_active"`
	EventSnapshotsActive    bool `yaml:"event_snapshots_active" json:"event_snapshots_active"`
	PeriodicSnapshotsActive bool `yaml:"periodic_snapshots_active" json:"periodic_snapshots_active"`
}

type OperatingStateStore interface {
	SaveOperatingState(streamName string, state OperatingState) error
}

func (state OperatingState) allowsLive() bool { return state.CameraActive && state.StreamingActive }
func (state OperatingState) allowsRecording() bool {
	return state.CameraActive && state.RecordingActive
}

func (s *Server) initOperatingState(cfg Config) {
	s.state = OperatingState{CameraActive: true, StreamingActive: true, EventSnapshotsActive: true, PeriodicSnapshotsActive: true}
	if cfg.OperatingState != nil {
		s.state = *cfg.OperatingState
	}
	if !cfg.HKSV {
		s.state.RecordingActive = false
	}
	s.stateStore = cfg.StateStore
	s.liveSessions = make(map[string]*liveConnTracker)
	for char, enabled := range s.operatingControls() {
		setControlValue(char, enabled)
	}
	s.updateOperatingStatus()
}

func (s *Server) operatingControls() map[*hap.Character]bool {
	values := make(map[*hap.Character]bool)
	if s.accessory == nil {
		return values
	}
	for _, entry := range []struct {
		service, typ string
		value        bool
	}{
		{"21A", "21B", s.state.CameraActive}, {"110", "B0", s.state.StreamingActive},
		{"204", "B0", s.state.RecordingActive}, {"204", "226", s.state.RecordingAudioActive},
		{"21A", "223", s.state.EventSnapshotsActive}, {"21A", "225", s.state.PeriodicSnapshotsActive},
	} {
		if service := s.accessory.GetService(entry.service); service != nil {
			if char := service.GetCharacter(entry.typ); char != nil {
				values[char] = entry.value
			}
		}
	}
	return values
}

func setControlValue(char *hap.Character, enabled bool) {
	if char.Format == hap.FormatBool {
		char.SetValue(enabled)
	} else if enabled {
		char.SetValue(1)
	} else {
		char.SetValue(0)
	}
}

func parseControl(value any) (bool, bool) {
	switch value := value.(type) {
	case bool:
		return value, true
	case float64:
		return value == 1, value == 0 || value == 1
	case int:
		return value == 1, value == 0 || value == 1
	}
	return false, false
}

// Called with controlMu held, the same lock used to start live and recording media.
func (s *Server) setOperatingControl(char *hap.Character, value any) (bool, int) {
	if _, ok := s.operatingControls()[char]; !ok {
		return false, 0
	}
	enabled, valid := parseControl(value)
	if !valid {
		return true, homekit.StatusInvalidValue
	}
	old := s.state
	switch char.Type {
	case "21B":
		s.state.CameraActive = enabled
	case "B0":
		if s.accessory.GetService("110").GetCharacter("B0") == char {
			s.state.StreamingActive = enabled
		} else {
			s.state.RecordingActive = enabled
		}
	case "226":
		s.state.RecordingAudioActive = enabled
	case "223":
		s.state.EventSnapshotsActive = enabled
	case "225":
		s.state.PeriodicSnapshotsActive = enabled
	}
	if old == s.state {
		return true, s.saveOperatingState()
	}
	setControlValue(char, enabled)
	s.updateOperatingStatus()
	if !s.state.allowsLive() {
		s.stopLiveStreamsLocked(nil)
	}
	// A changed audio setting requires a fresh MP4 init segment. Discard the old
	// recording and prebuffer so no buffered audio survives a mute operation.
	if !s.state.allowsRecording() || old.RecordingAudioActive != s.state.RecordingAudioActive {
		s.stopRecordingMedia()
	}
	if !s.state.CameraActive {
		s.motionMu.Lock()
		if char := s.accessory.GetCharacter("22"); char != nil {
			char.SetValue(false)
		}
		s.motionMu.Unlock()
	}
	if s.state.CameraActive && !old.CameraActive {
		switch s.motionMode {
		case "detect":
			go s.startMotionDetector()
		case "continuous":
			go s.startContinuousMotion()
		}
	}
	if s.state.allowsRecording() && (!old.allowsRecording() || old.RecordingAudioActive != s.state.RecordingAudioActive) {
		go s.prepareHKSVConsumer()
	}
	return true, s.saveOperatingState()
}

func (s *Server) saveOperatingState() int {
	if s.stateStore != nil {
		if err := s.stateStore.SaveOperatingState(s.stream, s.state); err != nil {
			s.log.Error().Err(err).Str("stream", s.stream).Msg("[hksv] save camera controls failed")
			return homekit.StatusCommunicationFailure
		}
	}
	return 0
}

func (s *Server) updateOperatingStatus() {
	if s.accessory == nil {
		return
	}
	if char := s.accessory.GetCharacter(camera.TypeStreamingStatus); char != nil {
		status := byte(camera.StreamingStatusUnavailable)
		if s.state.allowsLive() {
			status = camera.StreamingStatusAvailable
		}
		_ = char.Write(camera.StreamingStatus{Status: status})
	}
	if service := s.accessory.GetService("85"); service != nil {
		service.GetCharacter("75").SetValue(s.state.CameraActive)
	}
}

func (s *Server) cameraEnabled() bool {
	if s.accessory == nil {
		return false
	}
	char := s.accessory.GetCharacter("21B")
	if char == nil {
		return true
	} // Live-only legacy accessory.
	enabled, _ := parseControl(char.GetValue())
	return enabled
}

func (s *Server) SetCharacteristic(conn net.Conn, aid uint8, iid uint64, value any) {
	_ = s.SetCharacteristicStatus(conn, aid, iid, value)
}

func (s *Server) SetCharacteristicStatus(conn net.Conn, aid uint8, iid uint64, value any) int {
	s.controlMu.Lock()
	before := s.state
	status := s.writeCharacteristic(conn, aid, iid, value)
	after := s.state
	changed := before != after
	s.controlMu.Unlock()
	// Do not hold media/policy locks during event writes to other controllers.
	if changed {
		_ = s.accessory.GetCharacterByID(iid).NotifyListeners(conn)
		if before.allowsLive() != after.allowsLive() {
			_ = s.accessory.GetCharacter(camera.TypeStreamingStatus).NotifyListeners(conn)
		}
		if service := s.accessory.GetService("85"); service != nil && before.CameraActive != after.CameraActive {
			_ = service.GetCharacter("75").NotifyListeners(conn)
			if !after.CameraActive {
				_ = service.GetCharacter("22").NotifyListeners(conn)
			}
		}
	}
	return status
}

func (s *Server) stopRecordingMedia() {
	s.mu.Lock()
	sessions, prepared := slices.Collect(maps.Keys(s.recordingSessions)), s.preparedConsumer
	s.preparedConsumer = nil
	s.mu.Unlock()
	for _, session := range sessions {
		session.Close()
	}
	if prepared != nil {
		_ = prepared.Stop()
		s.streams.RemoveConsumer(s.stream, prepared)
	}
}

func (s *Server) newRecordingConsumer() *HKSVConsumer {
	consumer := NewHKSVConsumer(s.log)
	if !s.state.RecordingAudioActive {
		consumer.Medias = consumer.Medias[:1]
	}
	return consumer
}

// reason: -1 = unspecified (older controllers), 0 = periodic, 1 = event.
// https://developers.homebridge.io/HAP-NodeJS/enums/ResourceRequestReason.html
func (s *Server) GetImageWithReason(_ net.Conn, width, height, reason int) ([]byte, int) {
	s.controlMu.Lock()
	defer s.controlMu.Unlock()
	if !s.state.allowsLive() {
		return nil, homekit.StatusNotAllowed
	}
	switch reason {
	case -1:
		if !s.state.EventSnapshotsActive || !s.state.PeriodicSnapshotsActive {
			return nil, homekit.StatusInsufficientPrivileges
		}
	case 0:
		if !s.state.PeriodicSnapshotsActive {
			return nil, homekit.StatusNotAllowed
		}
	case 1:
		if !s.state.EventSnapshotsActive {
			return nil, homekit.StatusNotAllowed
		}
	default:
		return nil, homekit.StatusInvalidValue
	}
	if s.snapshots == nil {
		return nil, homekit.StatusCommunicationFailure
	}
	image, err := s.snapshots.GetSnapshot(s.stream, width, height)
	if err != nil || len(image) == 0 {
		return nil, homekit.StatusCommunicationFailure
	}
	return image, 0
}

// A live stream can end on timeout without a HAP End command.
type liveConnTracker struct {
	*Server
	sessionID string
	conn      net.Conn
}

func (t *liveConnTracker) DelConn(value any) {
	t.Server.DelConn(value)
	t.mu.Lock()
	if t.liveSessions[t.sessionID] == t {
		delete(t.liveSessions, t.sessionID)
	}
	t.mu.Unlock()
}

func (s *Server) rememberLiveStream(id string, conn net.Conn) *liveConnTracker {
	tracker := &liveConnTracker{Server: s, sessionID: id, conn: conn}
	s.mu.Lock()
	s.liveSessions[id] = tracker
	s.mu.Unlock()
	return tracker
}

func (s *Server) forgetLiveStream(id string) {
	s.mu.Lock()
	delete(s.liveSessions, id)
	s.mu.Unlock()
}

func (s *Server) stopLiveStreams(conn net.Conn) {
	s.controlMu.Lock()
	defer s.controlMu.Unlock()
	s.stopLiveStreamsLocked(conn)
}

func (s *Server) stopLiveStreamsLocked(conn net.Conn) {
	s.mu.Lock()
	var ids []string
	for id, owner := range s.liveSessions {
		if conn == nil || owner.conn == conn {
			ids = append(ids, id)
			delete(s.liveSessions, id)
		}
	}
	s.mu.Unlock()
	for _, id := range ids {
		if s.liveStream != nil {
			_ = s.liveStream.StopStream(id, s)
		}
	}
}
