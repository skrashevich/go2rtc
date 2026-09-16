package hap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitIIDExtendedTypesPreservePairings(t *testing.T) {
	legacy := &Service{Type: "110", Characters: []*Character{{Type: "114"}, {Type: "8043"}}}
	modern := &Service{Type: "8033", Characters: []*Character{{Type: "8043"}, {Type: "37"}}}
	second := &Service{Type: "8033", Characters: []*Character{{Type: "8043"}}}
	acc := &Accessory{AID: 1, Services: []*Service{legacy, modern, second}}
	require.NotPanics(t, acc.InitIID)
	require.Equal(t, uint64(0x11110000), legacy.IID)
	require.Equal(t, uint64(0x11110114), legacy.Characters[0].IID)

	seen := map[uint64]bool{}
	for _, service := range acc.Services {
		ids := []uint64{service.IID}
		for _, char := range service.Characters {
			ids = append(ids, char.IID)
		}
		for _, id := range ids {
			require.NotZero(t, id)
			require.Less(t, id, uint64(1<<53), "JSON number must remain exact")
			require.False(t, seen[id], "duplicate IID %x", id)
			seen[id] = true
		}
	}
	modernID, charID := modern.IID, modern.Characters[0].IID
	acc.Services = []*Service{modern, {Type: "8010"}, legacy, second}
	acc.InitIID()
	require.Equal(t, modernID, modern.IID, "unrelated services must not renumber IIDs")
	require.Equal(t, charID, modern.Characters[0].IID)
	require.Equal(t, uint64(0x11110114), legacy.Characters[0].IID)
}
