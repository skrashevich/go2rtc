package hksv

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCameraModeDoesNotSilentlyFallback(t *testing.T) {
	for _, mode := range []string{"secure_video", "typo"} {
		t.Run(mode, func(t *testing.T) {
			var cfg Config
			require.NoError(t, json.Unmarshal([]byte(`{"StreamName":"cam","HKSV":true,"Mode":"`+mode+`"}`), &cfg))
			srv, err := NewServer(cfg)
			require.Error(t, err, "unsupported mode must not publish a legacy accessory")
			require.Nil(t, srv)
			require.Contains(t, err.Error(), mode)
		})
	}
}

func TestLegacyModePreservesAccessory(t *testing.T) {
	for _, hksv := range []bool{false, true} {
		cfg := Config{StreamName: "cam", HKSV: hksv}
		defaultServer, err := NewServer(cfg)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal([]byte(`{"Mode":"legacy"}`), &cfg))
		explicitServer, err := NewServer(cfg)
		require.NoError(t, err)
		a, err := json.Marshal(defaultServer.Accessory())
		require.NoError(t, err)
		b, err := json.Marshal(explicitServer.Accessory())
		require.NoError(t, err)
		require.JSONEq(t, string(a), string(b))
	}
}
