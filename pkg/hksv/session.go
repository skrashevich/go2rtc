// Author: Sergei "svk" Krashevich <svk@svk.su>
package hksv

import (
	"sync"

	"github.com/AlexxIT/go2rtc/pkg/hap"
	"github.com/AlexxIT/go2rtc/pkg/hap/hds"
	"github.com/rs/zerolog"
)

// hksvSession manages the HDS DataStream connection for HKSV recording
type hksvSession struct {
	server  *Server
	hapConn *hap.Conn
	hdsConn *hds.Conn
	session *hds.Session
	log     zerolog.Logger

	mu       sync.Mutex
	consumer *HKSVConsumer
	closed   bool
}

func newHKSVSession(srv *Server, hapConn *hap.Conn, hdsConn *hds.Conn) *hksvSession {
	session := hds.NewSession(hdsConn)
	hs := &hksvSession{
		server:  srv,
		hapConn: hapConn,
		hdsConn: hdsConn,
		session: session,
		log:     srv.log,
	}
	session.CheckDataSendOpen = func() error {
		srv.controlMu.Lock()
		defer srv.controlMu.Unlock()
		if !srv.state.allowsRecording() {
			return hds.ErrNotAllowed
		}
		return nil
	}
	session.OnDataSendOpen = hs.handleOpen
	session.OnDataSendClose = hs.handleClose
	srv.mu.Lock()
	if srv.recordingSessions == nil {
		srv.recordingSessions = make(map[*hksvSession]bool)
	}
	srv.recordingSessions[hs] = true
	srv.mu.Unlock()
	return hs
}

func (hs *hksvSession) Run() error {
	return hs.session.Run()
}

func (hs *hksvSession) Close() {
	// Unblock any in-flight HDS write before waiting for the consumer lock.
	_ = hs.session.Close()
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.closed = true
	hs.server.mu.Lock()
	delete(hs.server.recordingSessions, hs)
	hs.server.mu.Unlock()
	if hs.consumer != nil {
		hs.stopRecording()
	}
	_ = hs.session.Close()
}

func (hs *hksvSession) handleOpen(streamID int) error {
	hs.server.controlMu.Lock()
	defer hs.server.controlMu.Unlock()
	if !hs.server.state.allowsRecording() {
		return hds.ErrNotAllowed
	}
	hs.mu.Lock()
	defer hs.mu.Unlock()

	hs.log.Debug().Str("stream", hs.server.stream).Int("streamID", streamID).Msg("[hksv] dataSend open")
	if hs.closed {
		return hds.ErrNotAllowed
	}

	if hs.consumer != nil {
		hs.stopRecording()
	}

	// Try to use the pre-started consumer from pair-verify
	consumer := hs.server.takePreparedConsumer()
	if consumer != nil {
		hs.log.Debug().Str("stream", hs.server.stream).Msg("[hksv] using prepared consumer")
		hs.consumer = consumer
		hs.server.AddConn(consumer)

		go func() {
			if err := consumer.Activate(hs.session, streamID); err != nil {
				hs.log.Debug().Err(err).Msg("[hksv] activate stopped")
			}
		}()
		return nil
	}

	// Fallback: create new consumer (will be slow ~3s)
	hs.log.Debug().Str("stream", hs.server.stream).Msg("[hksv] no prepared consumer, creating new")
	consumer = hs.server.newRecordingConsumer()

	if err := hs.server.streams.AddConsumer(hs.server.stream, consumer); err != nil {
		hs.log.Error().Err(err).Str("stream", hs.server.stream).Msg("[hksv] add consumer failed")
		return nil
	}

	hs.consumer = consumer
	hs.server.AddConn(consumer)

	go func() {
		if err := consumer.Activate(hs.session, streamID); err != nil {
			hs.log.Error().Err(err).Str("stream", hs.server.stream).Msg("[hksv] activate failed")
		}
	}()

	return nil
}

func (hs *hksvSession) handleClose(streamID int) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	hs.log.Debug().Str("stream", hs.server.stream).Int("streamID", streamID).Msg("[hksv] dataSend close")

	if hs.consumer != nil {
		hs.stopRecording()
	}
	return nil
}

func (hs *hksvSession) stopRecording() {
	consumer := hs.consumer
	hs.consumer = nil

	_ = consumer.Stop()
	hs.server.streams.RemoveConsumer(hs.server.stream, consumer)
	hs.server.DelConn(consumer)
}
