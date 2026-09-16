package hksv

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/homekit"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/hap/camera"
	"github.com/AlexxIT/go2rtc/pkg/hap/tlv8"
	"github.com/stretchr/testify/require"
)

func writeControl(t *testing.T, s *Server, service, typ string, value any) {
	t.Helper()
	c := s.accessory.GetService(service).GetCharacter(typ)
	require.NotNil(t, c)
	s.SetCharacteristic(nil, 1, c.IID, value)
}

func TestPrivacyStopsLiveAndBlocksRestart(t *testing.T) {
	live := &mockLiveStreamHandler{}
	s := newTestServer(t, func(c *Config) { c.LiveStream = live })
	start, err := tlv8.MarshalBase64(camera.SelectedStreamConfiguration{Control: camera.SessionControl{SessionID: "privacy-session!", Command: camera.SessionCommandStart}})
	require.NoError(t, err)
	c := s.accessory.GetCharacter(camera.TypeSelectedStreamConfiguration)
	s.SetCharacteristic(nil, 1, c.IID, start)
	require.True(t, live.startCalled)
	writeControl(t, s, "21A", "21B", false)
	require.True(t, live.stopCalled, "turning the camera off must stop its current live stream")
	live.startCalled = false
	s.SetCharacteristic(nil, 1, c.IID, start)
	require.False(t, live.startCalled, "off must block new streams")
	writeControl(t, s, "21A", "21B", true)
	s.SetCharacteristic(nil, 1, c.IID, start)
	require.True(t, live.startCalled, "turning on must restore live view")
}

func TestPrivacyBlocksSnapshotsAndMotion(t *testing.T) {
	snapshots := &mockSnapshotProvider{data: []byte("jpeg")}
	s := newTestServer(t, func(c *Config) { c.Snapshots = snapshots })
	s.SetMotionDetected(true)
	writeControl(t, s, "21A", "21B", false)
	require.False(t, s.MotionDetected())
	s.SetMotionDetected(true)
	require.False(t, s.MotionDetected(), "API and ONVIF motion must respect off")
	require.Empty(t, s.GetImage(nil, 320, 240))
	require.False(t, snapshots.called)
}

func TestStreamingDisabledKeepsMotion(t *testing.T) {
	snapshots := &mockSnapshotProvider{data: []byte("jpeg")}
	s := newTestServer(t, func(c *Config) { c.Snapshots = snapshots })
	writeControl(t, s, "110", "B0", float64(0))
	s.SetMotionDetected(true)
	require.True(t, s.MotionDetected(), "detect activity is independent of live streaming")
	require.Empty(t, s.GetImage(nil, 320, 240))
	require.False(t, snapshots.called)
}

func TestRecordingDisabledRejectsHDSOpen(t *testing.T) {
	hs, _, s := newTestHKSVSession(t, nil)
	writeControl(t, s, "204", "B0", float64(0))
	require.Error(t, hs.handleOpen(1))
	require.Nil(t, hs.consumer)
}

func TestSnapshotReasons(t *testing.T) {
	for _, test := range []struct {
		name            string
		event, periodic bool
		reason, status  int
	}{
		{"both enabled, older controller", true, true, -1, 0},
		{"event disabled", false, true, 1, homekit.StatusNotAllowed},
		{"periodic still enabled", false, true, 0, 0},
		{"periodic disabled", true, false, 0, homekit.StatusNotAllowed},
		{"event still enabled", true, false, 1, 0},
		{"missing reason cannot bypass policy", false, true, -1, homekit.StatusInsufficientPrivileges},
		{"unknown reason", true, true, 7, homekit.StatusInvalidValue},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshots := &mockSnapshotProvider{data: []byte("jpeg")}
			s := newTestServer(t, func(c *Config) { c.Snapshots = snapshots })
			writeControl(t, s, "21A", "223", test.event)
			writeControl(t, s, "21A", "225", test.periodic)
			image, status := s.GetImageWithReason(nil, 320, 240, test.reason)
			require.Equal(t, test.status, status)
			require.Equal(t, status == 0, snapshots.called)
			require.Equal(t, status == 0, len(image) > 0)
		})
	}
}

func TestRecordingAudioChangesMP4Tracks(t *testing.T) {
	s := newTestServer(t)
	for _, audio := range []bool{false, true} {
		writeControl(t, s, "204", "226", audio)
		consumer := s.newRecordingConsumer()
		for _, media := range consumer.Medias {
			codec := &core.Codec{Name: core.CodecH264, ClockRate: 90000}
			if media.Kind == core.KindAudio {
				codec = &core.Codec{Name: core.CodecAAC, ClockRate: 24000, Channels: 1}
			}
			require.NoError(t, consumer.AddTrack(media, codec, core.NewReceiver(media, codec)))
		}
		// Inspect the actual MP4 init segment, not only the advertised media list.
		require.Contains(t, string(consumer.initData), "avc1")
		require.Equal(t, audio, bytes.Contains(consumer.initData, []byte("mp4a")))
		require.NoError(t, consumer.Stop())
	}
}

func TestOffStopsRecordingAndPreparedMedia(t *testing.T) {
	hs, ctrl, s := newTestHKSVSession(t, nil)
	go func() {
		for {
			if _, err := ctrl.ReadMessage(); err != nil {
				return
			}
		}
	}()
	require.NoError(t, hs.handleOpen(1))
	active := hs.consumer
	prepared := s.newRecordingConsumer()
	require.NoError(t, s.streams.AddConsumer(s.stream, prepared))
	s.mu.Lock()
	s.preparedConsumer = prepared
	s.mu.Unlock()
	writeControl(t, s, "21A", "21B", false)
	select {
	case <-active.Done():
	default:
		t.Fatal("recording still active")
	}
	select {
	case <-prepared.Done():
	default:
		t.Fatal("prepared media still active")
	}
	require.Zero(t, s.streams.(*mockStreamProvider).count(s.stream))
	require.Nil(t, hs.consumer)
	// A late activation must never reactivate a stopped consumer.
	require.Error(t, active.Activate(hs.session, 2))
	require.False(t, active.active)
}

func TestInvalidControlDoesNotChangePolicy(t *testing.T) {
	s := newTestServer(t)
	char := s.accessory.GetCharacter("21B")
	for _, value := range []any{"false", float64(2), float64(0.5), nil} {
		require.Equal(t, homekit.StatusInvalidValue, s.SetCharacteristicStatus(nil, 1, char.IID, value))
		require.Equal(t, true, char.GetValue())
	}
}

// Exercise HTTP parsing, per-characteristic errors and snapshot reason forwarding
// through the real HAP handler while keeping the connection open across requests.
func TestPrivacyHTTP(t *testing.T) {
	snapshots := &mockSnapshotProvider{data: []byte("jpeg")}
	s := newTestServer(t, func(c *Config) { c.Snapshots = snapshots })
	accessory, controller := net.Pipe()
	t.Cleanup(func() { accessory.Close(); controller.Close() })
	require.NoError(t, controller.SetDeadline(time.Now().Add(3*time.Second)))
	go func() { _ = homekit.ServerHandler(s)(accessory) }()
	reader := bufio.NewReader(controller)
	request := func(method, path string, body any) (*http.Response, []byte) {
		t.Helper()
		data, err := json.Marshal(body)
		require.NoError(t, err)
		req, err := http.NewRequest(method, "http://camera"+path, bytes.NewReader(data))
		require.NoError(t, err)
		require.NoError(t, req.Write(controller))
		res, err := http.ReadResponse(reader, req)
		require.NoError(t, err)
		data, err = io.ReadAll(res.Body)
		require.NoError(t, err)
		res.Body.Close()
		return res, data
	}
	event := s.accessory.GetCharacter("223")
	res, _ := request("PUT", "/characteristics", map[string]any{"characteristics": []any{map[string]any{"aid": 1, "iid": event.IID, "value": false, "r": true}}})
	require.Equal(t, http.StatusOK, res.StatusCode)
	res, body := request("POST", "/resource", map[string]any{"resource-type": "image", "reason": 0})
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "jpeg", string(body))
	res, body = request("POST", "/resource", map[string]any{"resource-type": "image", "reason": 1})
	require.Equal(t, http.StatusBadRequest, res.StatusCode)
	require.Contains(t, string(body), "-70412")
	cameraActive := s.accessory.GetCharacter("21B")
	res, _ = request("PUT", "/characteristics", map[string]any{"characteristics": []any{map[string]any{"aid": 1, "iid": cameraActive.IID, "value": false}}})
	require.Equal(t, http.StatusNoContent, res.StatusCode)
	setup := s.accessory.GetCharacter(camera.TypeSetupEndpoints)
	res, body = request("PUT", "/characteristics", map[string]any{"characteristics": []any{map[string]any{"aid": 1, "iid": setup.IID, "value": ""}}})
	require.Equal(t, http.StatusMultiStatus, res.StatusCode)
	require.Contains(t, string(body), "-70412")
	res, body = request("POST", "/resource", map[string]any{"resource-type": "image", "reason": 0})
	require.Equal(t, http.StatusBadRequest, res.StatusCode)
	require.Contains(t, string(body), "-70412")
}
