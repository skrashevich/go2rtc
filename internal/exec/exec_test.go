package exec

import (
	"bytes"
	"testing"
)

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input    []byte
		expected []byte
	}{
		{[]byte("   Hello, World!   "), []byte("Hello, World!")},
		{[]byte("Hello, World!"), []byte("Hello, World!")},
		{[]byte("     "), nil},
		{[]byte(""), nil},
		{[]byte("\t\n Hello, \n\tWorld! \t\n"), []byte("Hello, \n\tWorld!")},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := trimSpace(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("trimSpace(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func BenchmarkTrimSpace(b *testing.B) {
	input := []byte("\t\n Hello, \n\tWorld! \t\n")
	for i := 0; i < b.N; i++ {
		_ = trimSpace(input)
	}
}
