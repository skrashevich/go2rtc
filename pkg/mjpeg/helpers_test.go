package mjpeg

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"testing"
)

// helper function to create a simple JPEG image in memory
func createTestJPEG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	buf := bytes.NewBuffer(nil)
	jpeg.Encode(buf, img, nil)
	return buf.Bytes()
}
func TestFixJPEG(t *testing.T) {
	// Test non-JPEG data
	nonJPEG := []byte("not a jpeg")
	if got := FixJPEG(nonJPEG); !bytes.Equal(got, nonJPEG) {
		t.Errorf("FixJPEG(nonJPEG) = %v, want %v", got, nonJPEG)
	}

	// Test valid JPEG data
	validJPEG := createTestJPEG()
	fixedJPEG := FixJPEG(validJPEG)
	if got := FixJPEG(validJPEG); !bytes.Equal(got, validJPEG) {
		t.Errorf("FixJPEG(validJPEG) should not alter the input")
	}
	// Optionally, verify the fixed JPEG starts with the correct JPEG SOI marker
	if len(fixedJPEG) < 2 || fixedJPEG[0] != 0xFF || fixedJPEG[1] != 0xD8 {
		t.Error("Fixed JPEG does not start with the standard JPEG SOI marker")
	}

	// Test non-JPEG data
	nonJPEG = makeBadJPEG()
	fixedJPEG = FixJPEG(nonJPEG)
	if !bytes.Equal(fixedJPEG, nonJPEG) {
		t.Error("FixJPEG(nonJPEG) did not alter a bad JPEG as expected")
	}

}
func BenchmarkFixJPEG(b *testing.B) {
	// Use a representative JPEG byte slice for benchmarking
	jpegData := createTestJPEG() // Assuming this creates a "bad" JPEG for fixing

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FixJPEG(jpegData)
	}
}

// Helper function to create a "bad" JPEG byte slice.
// This is a simplistic approach; adjust according to what constitutes a "bad" JPEG in your context.
func makeBadJPEG() []byte {
	// Start with a valid JPEG and intentionally corrupt the header
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	buf := bytes.NewBuffer(nil)
	png.Encode(buf, img)
	jpegBytes := buf.Bytes()

	// Corrupting the JPEG by changing its SOI marker to something else
	if len(jpegBytes) > 2 {
		jpegBytes[0] = 0x00
		jpegBytes[1] = 0x00
	}
	return jpegBytes
}
