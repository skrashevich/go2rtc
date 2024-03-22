package homekit

import (
	"testing"
)

// calcDeviceSerial remains unchanged, placed here for context.

func TestCalcDeviceSerial(t *testing.T) {
	tests := []struct {
		name   string
		serial string
		seed   string
		want   string // Expected outcome; for dynamic outcomes, this might need to be adjusted.
	}{
		{
			name:   "Valid serial, meets criteria",
			serial: "123456789012",
			seed:   "unusedSeed",
			want:   "123456789012",
		},
		{
			name:   "Serial as seed",
			serial: "short",
			seed:   "defaultSeed",
			want:   "", // This would be dynamically generated based on the 'short' seed.
		},
		{
			name:   "Empty serial, use seed",
			serial: "",
			seed:   "mySeedString",
			want:   "", // Dynamically generated based on 'mySeedString'.
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := calcDeviceSerial(tc.serial, tc.seed)
			if tc.want == "" {
				// For dynamic outcomes, we might want to check the format instead of exact value.
				if len(got) == 0 || got == tc.seed {
					t.Errorf("calcDeviceSerial(%q, %q) = %q, want non-empty and not equal to seed", tc.serial, tc.seed, got)
				}
			} else if got != tc.want {
				t.Errorf("calcDeviceSerial(%q, %q) = %q, want %q", tc.serial, tc.seed, got, tc.want)
			}
		})
	}
}
