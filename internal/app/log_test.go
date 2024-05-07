package app

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestCircularBuffer(t *testing.T) {
	buf := newBuffer(2) // Small buffer for testing

	// Test writing and wrapping
	msg1 := []byte("hello")
	msg2 := []byte("world")
	_, err := buf.Write(msg1)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	_, err = buf.Write(msg2)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	// Test buffer content
	expected := "helloworld"
	if string(buf.Bytes()) != expected {
		t.Errorf("Expected %s, got %s", expected, string(buf.Bytes()))
	}

	// Test reset
	buf.Reset()
	if len(buf.Bytes()) != 0 {
		t.Errorf("Expected empty buffer after reset, got %d bytes", len(buf.Bytes()))
	}
}
func TestNewLogger(t *testing.T) {
	tests := []struct {
		format string
		level  string
	}{
		{"json", "info"},
		{"text", "debug"},
	}

	for _, tc := range tests {
		logger := NewLogger(tc.format, tc.level)

		// Check if logger has the correct level
		lvl := logger.GetLevel()
		expectedLvl, _ := zerolog.ParseLevel(tc.level)
		if lvl != expectedLvl {
			t.Errorf("Expected level %s, got %s", tc.level, lvl.String())
		}

		// Additional checks can be added here for format verification
	}
}
func TestGetLogger(t *testing.T) {
	modules = map[string]string{
		"module1": "debug",
		"module2": "warn",
	}

	logger1 := GetLogger("module1")
	if logger1.GetLevel() != zerolog.DebugLevel {
		t.Errorf("Expected debug level for module1, got %s", logger1.GetLevel().String())
	}

	logger2 := GetLogger("module2")
	if logger2.GetLevel() != zerolog.WarnLevel {
		t.Errorf("Expected warn level for module2, got %s", logger2.GetLevel().String())
	}

	// Test non-existent module (should default to global logger level)
	logger3 := GetLogger("nonexistent")
	if logger3.GetLevel() != log.Logger.GetLevel() {
		t.Errorf("Expected default logger level for nonexistent module, got %s", logger3.GetLevel().String())
	}
}
