package homekit

import (
	"strings"
	"testing"
)

// TestCalcDeviceSerialBasicFunctionality checks if the function returns the expected serial for known inputs.
func TestCalcDeviceSerialBasicFunctionality(t *testing.T) {
	tests := []struct {
		currentSerial string
		seed          string
		want          string
	}{
		{"1234567890ABCDEF", "seed456", "1234567890ABCDEF"},
		{"longSerialNumberWithVariousCharacters", "complexSeedValue", "9J300369071M9695"},
		{"3171M89JM00530PP", "kitchen", "3171M89JM00530PP"},
	}

	for _, tt := range tests {
		got := calcDeviceSerial(tt.currentSerial, tt.seed)
		if got != tt.want {
			t.Errorf("calcDeviceSerial(%q, %q) got  %s, want %s", tt.currentSerial, tt.seed, got, tt.want)
		}
	}
}

// TestCalcDeviceSerialOutputConsistency checks whether the function consistently produces the same output for the same input.
func TestCalcDeviceSerialOutputConsistency(t *testing.T) {
	currentSerial := "consistentSerial"
	seed := "consistentSeed"
	expected := calcDeviceSerial(currentSerial, seed)

	for i := 0; i < 10; i++ {
		result := calcDeviceSerial(currentSerial, seed)
		if result != expected {
			t.Errorf("Inconsistent result for the same input: got %s, want %s", result, expected)
		}
	}
}

// TestCalcDeviceSerialAmbiguousCharacters verifies that the output does not contain any ambiguous characters.
func TestCalcDeviceSerialAmbiguousCharacters(t *testing.T) {
	currentSerial := "testSerial"
	seed := "testSeed"
	ambiguousChars := "abcdef"

	result := calcDeviceSerial(currentSerial, seed)

	for _, c := range ambiguousChars {
		if strings.ContainsRune(result, c) {
			t.Errorf("Found ambiguous character %c in result %s", c, result)
		}
	}
}

// TestCalcDeviceSerialLength ensures the serial number is always exactly 16 characters long.
func TestCalcDeviceSerialLength(t *testing.T) {
	tests := []struct {
		currentSerial string
		seed          string
	}{
		{"short", "seed"},
		{"", ""},
		{"veryVeryLongSerialNumber", "withALongSeedAsWell"},
	}

	const expectedLength = 16

	for _, tt := range tests {
		result := calcDeviceSerial(tt.currentSerial, tt.seed)
		if len(result) != expectedLength {
			t.Errorf("calcDeviceSerial(%q, %q) resulted in length %d, want %d", tt.currentSerial, tt.seed, len(result), expectedLength)
		}
	}
}

func Test_calcDeviceSerial(t *testing.T) {
	type args struct {
		currentSerial string
		seed          string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "kitchen", args: args{currentSerial: "", seed: "kitchen"}, want: "3171M89JM00530PP"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calcDeviceSerial(tt.args.currentSerial, tt.args.seed); got != tt.want {
				t.Errorf("calcDeviceSerial() = %v, want %v", got, tt.want)
			}
		})
	}
}
