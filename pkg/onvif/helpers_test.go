package onvif // change this to your actual package name

import (
	"regexp"
	"testing"
)

// TestUUIDFormat checks if the UUID function generates strings in the expected format
func TestUUIDFormat(t *testing.T) {
	uuid := UUID()
	expectedPattern := regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

	if !expectedPattern.MatchString(uuid) {
		t.Errorf("Generated UUID '%s' does not match the expected format", uuid)
	}
}

// TestUUIDLength checks if the UUID function generates strings of the correct length
func TestUUIDLength(t *testing.T) {
	uuid := UUID()
	expectedLength := 36 // UUID format 8-4-4-4-12

	if len(uuid) != expectedLength {
		t.Errorf("Generated UUID '%s' does not have the expected length of %d", uuid, expectedLength)
	}
}

// TestUUIDUniqueness checks if multiple invocations of UUID generate unique values
func TestUUIDUniqueness(t *testing.T) {
	uuids := make(map[string]bool)
	iterations := 10000

	for i := 0; i < iterations; i++ {
		uuid := UUID()
		if _, exists := uuids[uuid]; exists {
			t.Errorf("Generated UUID '%s' is not unique", uuid)
		}
		uuids[uuid] = true
	}
}
