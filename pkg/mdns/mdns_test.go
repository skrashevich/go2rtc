package mdns

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDiscovery(t *testing.T) {
	onentry := func(entry *ServiceEntry) bool {
		return true
	}
	err := Discovery(ServiceHAP, onentry)
	//err := Discovery("_ewelink._tcp.local.", time.Second, onentry)
	// err := Discovery("_googlecast._tcp.local.", time.Second, onentry)
	assert.Nil(t, err)
}
