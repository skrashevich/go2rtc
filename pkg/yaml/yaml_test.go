package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatch(t *testing.T) {
	b := []byte(`# prefix`)

	// 1. Add first
	b, err := Patch(b, "camera1", "url1", "streams")
	assert.Nil(t, err)

	assert.Equal(t, `# prefix
streams:
  camera1: url1
`, string(b))

	// 2. Add second
	b, err = Patch(b, "camera2", []string{"url2", "url3"}, "streams")
	assert.Nil(t, err)

	assert.Equal(t, `# prefix
streams:
  camera1: url1
  camera2:
    - url2
    - url3
`, string(b))

	// 3. Replace first
	b, err = Patch(b, "camera1", "url4", "streams")
	assert.Nil(t, err)

	assert.Equal(t, `# prefix
streams:
  camera1: url4
  camera2:
    - url2
    - url3
`, string(b))

	// 4. Replace second
	b, err = Patch(b, "camera2", "url5", "streams")
	assert.Nil(t, err)

	assert.Equal(t, `# prefix
streams:
  camera1: url4
  camera2: url5
`, string(b))

	// 5. Delete first
	b, err = Patch(b, "camera1", nil, "streams")
	assert.Nil(t, err)

	assert.Equal(t, `# prefix
streams:
  camera2: url5
`, string(b))
}

func TestPatchParings(t *testing.T) {
	b := []byte(`homekit:
  camera1:
    pin: 123-45-678
streams:
  camera1: url1
`)

	// 1. Add new key
	pairings := []string{"client1", "client2"}

	b, err := Patch(b, "pairings", pairings, "homekit", "camera1")
	assert.Nil(t, err)

	assert.Equal(t, `homekit:
  camera1:
    pin: 123-45-678
    pairings:
      - client1
      - client2
streams:
  camera1: url1
`, string(b))
}

func TestPatch2(t *testing.T) {
	b := []byte(`streams:
  camera1:
    - url1
    - url2
`)

	b, err := Patch(b, "camera2", "url3", "streams")
	assert.Nil(t, err)

	assert.Equal(t, `streams:
  camera1:
    - url1
    - url2
  camera2: url3
`, string(b))
}

func TestNoNewLineEnd1(t *testing.T) {
	b := []byte(`streams:
  camera1: url4
  camera2:
    - url2
    - url3`)

	b, err := Patch(b, "camera2", "url5", "streams")
	assert.Nil(t, err)

	assert.Equal(t, `streams:
  camera1: url4
  camera2: url5
`, string(b))
}

func TestNoNewLineEnd2(t *testing.T) {
	b := []byte(`streams:
  camera1: url1
homekit:
  camera1:
    pin: 123-45-678`)

	// 1. Add new key
	pairings := []string{"client1", "client2"}

	b, err := Patch(b, "pairings", pairings, "homekit", "camera1")
	assert.Nil(t, err)

	assert.Equal(t, `streams:
  camera1: url1
homekit:
  camera1:
    pin: 123-45-678
    pairings:
      - client1
      - client2
`, string(b))
}
